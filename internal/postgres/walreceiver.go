package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgproto3/v2"
)

const (
	defaultTimeout = time.Second * 10
	streamNameCol  = "stream_name"
	streamIDCol    = "stream_id"
	eventNameCol   = "event_name"
	versionCol     = "version"
	rawDataCol     = "raw_data"
)

type WalData struct {
	StreamName string
	StreamID   string
	EventName  string
	Version    int64
	Data       []byte
}

func (d WalData) Validate() error {
	if len(d.StreamName) == 0 && len(d.StreamID) == 0 || len(d.Data) == 0 {
		return fmt.Errorf("invalid data schema gulfstream.outbox")
	}
	return nil
}

type WalReceiver struct {
	ctx                  context.Context
	conn                 *pgconn.PgConn
	slotName             string
	timeout              time.Duration
	primaryKeepaliveFunc []func(pglogrepl.PrimaryKeepaliveMessage)
	standbyStatusFunc    []func(pglogrepl.LSN)
	messageFunc          []HandleFunc
	errorFunc            []func(*pgproto3.ErrorResponse)
	relations            map[uint32]*pglogrepl.RelationMessage
	exitFunc             []func()
	beginTxPos           pglogrepl.LSN
	commitTxPos          pglogrepl.LSN
	confirmTxPos         pglogrepl.LSN
}

type HandleFunc func(data WalData) error

func NewWalReceiverWithContext(
	ctx context.Context,
	conn *pgconn.PgConn,
	slotName string,
) *WalReceiver {
	return &WalReceiver{
		ctx:       ctx,
		timeout:   defaultTimeout,
		conn:      conn,
		slotName:  slotName,
		relations: make(map[uint32]*pglogrepl.RelationMessage),
	}
}

func (r *WalReceiver) OnExit(fn func()) {
	r.exitFunc = append(r.exitFunc, fn)
}

func (r *WalReceiver) OnError(fn func(*pgproto3.ErrorResponse)) {
	r.errorFunc = append(r.errorFunc, fn)
}

func (r *WalReceiver) OnPrimaryKeepalive(fn func(pglogrepl.PrimaryKeepaliveMessage)) {
	r.primaryKeepaliveFunc = append(r.primaryKeepaliveFunc, fn)
}

func (r *WalReceiver) OnStandbyStatusUpdate(fn func(pglogrepl.LSN)) {
	r.standbyStatusFunc = append(r.standbyStatusFunc, fn)
}

func (r *WalReceiver) OnMessage(fn HandleFunc) {
	r.messageFunc = append(r.messageFunc, fn)
}

func (r *WalReceiver) Listen() error {
	lastLSN, err := r.lastLSN()
	if err != nil {
		return err
	}
	if err := pglogrepl.StartReplication(r.ctx, r.conn, r.slotName, lastLSN,
		pglogrepl.StartReplicationOptions{
			Mode: pglogrepl.LogicalReplication,
			PluginArgs: []string{
				"proto_version '1'",
				"publication_names '" + r.slotName + "'",
			},
		}); err != nil {
		return err
	}

	r.beginTxPos = 0
	r.commitTxPos = 0
	r.confirmTxPos = lastLSN
	nextDeadline := time.Now().Add(r.timeout)

	defer func() {
		for _, fn := range r.exitFunc {
			fn()
		}
	}()

	for r.ctx.Err() == nil {
		if time.Now().After(nextDeadline) {
			if err := pglogrepl.SendStandbyStatusUpdate(r.ctx, r.conn,
				pglogrepl.StandbyStatusUpdate{
					WALWritePosition: r.confirmTxPos,
				}); err != nil {
				return err
			}
			nextDeadline = time.Now().Add(r.timeout)
			for _, fn := range r.standbyStatusFunc {
				fn(r.confirmTxPos)
			}
		}

		ctx, cancel := context.WithDeadline(r.ctx, nextDeadline)
		msg, err := r.conn.ReceiveMessage(ctx)
		cancel()
		if err != nil {
			if pgconn.Timeout(err) {
				if r.ctx.Err() != nil {
					return nil
				}
				continue
			}
			return err
		}

		switch msg := msg.(type) {
		case *pgproto3.ErrorResponse:
			for _, fn := range r.errorFunc {
				fn(msg)
			}
		case *pgproto3.CopyData:
			switch msg.Data[0] {
			case pglogrepl.PrimaryKeepaliveMessageByteID:
				pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
				if err != nil {
					return fmt.Errorf("postgres: WalReceiver.SendStandbyStatusUpdate failed: %w", err)
				}
				if pkm.ReplyRequested {
					nextDeadline = time.Time{}
				}
				for _, fn := range r.primaryKeepaliveFunc {
					fn(pkm)
				}

			case pglogrepl.XLogDataByteID:
				xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
				if err != nil {
					return fmt.Errorf("postgres: WalReceiver.ParseXLogData failed: %w", err)
				}
				logicalMsg, err := pglogrepl.Parse(xld.WALData)
				if err != nil {
					return fmt.Errorf("postgres: WalReceiver.ParseWALData logical replication message: %w", err)
				}
				if err := r.handleMessage(logicalMsg); err != nil {
					return err
				}
				r.confirmTxPos = xld.WALStart + pglogrepl.LSN(len(xld.WALData))
			}
		}
	}
	return nil
}

func (r *WalReceiver) handleMessage(m pglogrepl.Message) (err error) {
	switch message := m.(type) {
	case *pglogrepl.BeginMessage:
		r.beginTxPos = message.FinalLSN
	case *pglogrepl.CommitMessage:
		r.commitTxPos = message.CommitLSN
		if r.commitTxPos != r.beginTxPos {
			return fmt.Errorf("postgres: WalReceiver.HandleMessage mismatch wal positions begin:%s, commit:%s",
				r.beginTxPos, r.commitTxPos)
		}
	case *pglogrepl.RelationMessage:
		r.relations[message.RelationID] = message
	case *pglogrepl.InsertMessage:
		rel, ok := r.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("postgres: WalReceiver could not find relation for row relid=%d, relname=%s",
				rel.RelationID, rel.RelationName)
		}
		var data WalData
		for i, c := range message.Tuple.Columns {
			switch rel.Columns[i].Name {
			case eventNameCol:
				data.EventName = string(c.Data)
			case streamIDCol:
				data.StreamID = string(c.Data)
			case streamNameCol:
				data.StreamName = string(c.Data)
			case versionCol:
				data.Version, err = c.Int64()
			case rawDataCol:
				data.Data = c.Data
			}
		}
		for _, fn := range r.messageFunc {
			if err := fn(data); err != nil {
				return err
			}
		}
	}
	return
}

func (r *WalReceiver) lastLSN() (lsn pglogrepl.LSN, err error) {
	reader := r.conn.Exec(r.ctx, "SELECT restart_lsn FROM pg_replication_slots WHERE slot_name='"+r.slotName+"'")
	res, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}
	if len(res) != 1 {
		return 0, fmt.Errorf("postgres: WalReceiver.LastLSN expected 1 resultSet, got %d", len(res))
	}
	if len(res[0].Rows) != 1 {
		return 0, fmt.Errorf("postgres: WalReceiver.LastLSN expected 1 result row, got %d", len(res[0].Rows))
	}
	return pglogrepl.ParseLSN(string(res[0].Rows[0][0]))
}
