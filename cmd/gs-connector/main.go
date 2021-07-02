package main

import (
	"fmt"
	"os"

	"github.com/go-gulfstream/connector/internal/commands"
)

func main() {
	app, err := commands.New()
	if err != nil {
		fmt.Printf("[ERROR] main.New %s\n", err)
		os.Exit(1)
	}
	if err := app.Execute(); err != nil {
		fmt.Printf("[ERROR] main.Execute %s\n", err)
		os.Exit(1)
	}
}

/*
connURI := "postgres://postgres:123456@127.0.0.1:5432/postgres?replication=database"
	ctx := context.Background()
	conn, err := pgconn.Connect(ctx, connURI)
	if err != nil {
		log.Fatalln("failed to connect to PostgreSQL server:", err)
	}
	defer conn.Close(ctx)

	walRecv := postgres.NewWalReceiverWithContext(ctx, conn, "my")
	walRecv.OnPrimaryKeepalive(func(message pglogrepl.PrimaryKeepaliveMessage) {
		log.Println("PrimaryKeepalive", message.ServerWALEnd)
	})
	walRecv.OnStandbyStatusUpdate(func(lsn pglogrepl.LSN) {
		log.Println("OnStandbyStatus", lsn)
	})
	walRecv.OnMessage(func(streamName string, streamID string, version int64, rawData []byte) error {
		log.Println(streamName, streamName, version, string(rawData))
		return nil
	})
	if err := walRecv.Listen(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
*/
