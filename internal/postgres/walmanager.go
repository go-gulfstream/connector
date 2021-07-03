package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pglogrepl"

	"github.com/jackc/pgconn"
)

const pgOutputPlugin = "pgoutput"

func DropSlot(ctx context.Context, conn *pgconn.PgConn, slotName string) error {
	resPub := conn.Exec(context.Background(), "DROP PUBLICATION IF EXISTS "+slotName+";")
	_, err := resPub.ReadAll()
	if err != nil {
		return err
	}
	if err := pglogrepl.DropReplicationSlot(ctx, conn, slotName,
		pglogrepl.DropReplicationSlotOptions{
			Wait: true,
		}); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42704" {
			return nil
		}
		return err
	}
	return nil
}

func CreateSlot(ctx context.Context, conn *pgconn.PgConn, slotName string) error {
	resPub := conn.Exec(context.Background(), "CREATE PUBLICATION "+slotName+" FOR TABLE gulfstream.outbox WITH (publish='insert');")
	_, err := resPub.ReadAll()
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if !ok {
			return err
		}
		if ok && pgErr.Code != "42710" {
			return err
		}
	}
	res, err := pglogrepl.CreateReplicationSlot(
		ctx,
		conn,
		slotName,
		pgOutputPlugin,
		pglogrepl.CreateReplicationSlotOptions{
			Mode:           pglogrepl.LogicalReplication,
			SnapshotAction: "NOEXPORT_SNAPSHOT",
		},
	)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42710" {
			return nil
		}
		return err
	}
	if res.SlotName != slotName {
		return fmt.Errorf("slot creation error got %s, expected %s",
			slotName, res.SlotName)
	}
	return nil
}
