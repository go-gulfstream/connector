package postgres

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgproto3/v2"
)

const (
	defaultTimeout = time.Second * 15
	streamNameCol  = "stream_name"
	streamIDCol    = "stream_id"
	versionCol     = "version"
	rawDataCol     = "raw_data"
)

type WalReceiver struct {
	ctx                  context.Context
	conn                 *pgconn.PgConn
	slotName             string
	timeout              time.Duration
	primaryKeepaliveFunc []func(pglogrepl.PrimaryKeepaliveMessage)
	standbyStatusFunc    []func(pglogrepl.LSN)
	messageFunc          []HandleFunc
	relations            map[uint32]*pglogrepl.RelationMessage
	beginTxPos           pglogrepl.LSN
	commitTxPos          pglogrepl.LSN
	confirmTxPos         pglogrepl.LSN
	streamName           string
	streamID             string
	version              int64
	rawData              *bytes.Buffer
}

type HandleFunc func(streamName string, streamID string, version int64, rawData []byte) error

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
		rawData:   bytes.NewBuffer(nil),
		relations: make(map[uint32]*pglogrepl.RelationMessage),
	}
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
				continue
			}
			return err
		}

		switch msg := msg.(type) {
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

		for _, fn := range r.messageFunc {
			if err := fn(r.streamName, r.streamID, r.version, r.rawData.Bytes()); err != nil {
				return err
			}
		}

		r.streamName = ""
		r.streamID = ""
		r.version = 0
		r.rawData.Reset()

	case *pglogrepl.RelationMessage:
		r.relations[message.RelationID] = message
	case *pglogrepl.InsertMessage:
		rel, ok := r.relations[message.RelationID]
		if !ok {
			return fmt.Errorf("postgres: WalReceiver could not find relation for row relid=%d, relname=%s",
				rel.RelationID, rel.RelationName)
		}
		for i, c := range message.Tuple.Columns {
			switch rel.Columns[i].Name {
			case streamIDCol:
				r.streamID = string(c.Data)
			case streamNameCol:
				r.streamName = string(c.Data)
			case versionCol:
				r.version, err = c.Int64()
			case rawDataCol:
				r.rawData.Write(c.Data)
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
		return 0, fmt.Errorf("postgres: WalReceiver.lastLSN expected 1 resultSet, got %d", len(res))
	}
	if len(res[0].Rows) != 1 {
		return 0, fmt.Errorf("postgres: WalReceiver.lastLSN expected 1 result row, got %d", len(res[0].Rows))
	}
	if len(res[0].Rows) != 1 {
		return 0, fmt.Errorf("postgres: WalReceiver.lastLSN expected 2 result columns, got %d", len(res[0].Rows))
	}
	return pglogrepl.ParseLSN(string(res[0].Rows[0][0]))
}
