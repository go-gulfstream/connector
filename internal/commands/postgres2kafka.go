package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgproto3/v2"

	"github.com/go-gulfstream/connector/internal/postgres"

	"github.com/jackc/pgconn"

	"github.com/Shopify/sarama"

	"github.com/go-gulfstream/connector/internal/config"
	"github.com/spf13/cobra"
)

type postgres2kafkaFlags struct {
	configFile string
	renewSlot  bool
	dropSlot   bool
	showConfig bool
}

func postgres2kafkaCommand() *cobra.Command {
	var flags postgres2kafkaFlags
	cmd := &cobra.Command{
		Use:   "postgres2kafka",
		Short: "Exporting gulfstream-events from postgres to kafka",
		Example: `
gs-connect postgres2kafka --config path/to/config.yml
gs-connect p2k -c path/to/config.yml --show-config
gs-connect p2k --drop-slot 
gs-connect p2k --renew-slot
`,
		Aliases:    []string{"p2k", "pg2kfk"},
		SuggestFor: []string{"p2", "post", "postgres", "pg"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := parseConfig(flags.configFile)
			if err != nil {
				return err
			}
			if err := config.Validate(cfg.Kafka, cfg.Postgres); err != nil {
				return err
			}
			ctx, cancel := context.WithCancel(context.Background())
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigs
				cancel()
			}()
			return runPostgres2kafkaCommand(ctx, cfg, flags)
		},
	}
	cmd.Flags().StringVarP(&flags.configFile, "config", "c", "", "config file (default is $PWD/gc-connector.yaml)")
	cmd.Flags().BoolVarP(&flags.renewSlot, "renew-slot", "r", false, "Renew slot")
	cmd.Flags().BoolVarP(&flags.dropSlot, "drop-slot", "d", false, "Drop slot")
	cmd.Flags().BoolVarP(&flags.showConfig, "show-config", "s", false, "Show current configuration")
	return cmd
}

func runPostgres2kafkaCommand(ctx context.Context, cfg *config.Config, flags postgres2kafkaFlags) error {
	kafkaConfig := cfg.Kafka.NewConfig()

	if flags.showConfig {
		fmt.Printf("====================== postgres2kafka %s ======================\n", flags.configFile)
		fmt.Printf("Postgres:\n")
		fmt.Printf("  ConnectionURI: %s\n", cfg.Postgres.ConnectionURI)
		fmt.Printf("  SlotName: %s\n", cfg.Postgres.GetSlotName())
		fmt.Printf("Kafka:\n")
		fmt.Printf("  Brokers: %s\n", cfg.Kafka.Brokers)
		fmt.Printf("  ClientID: %s\n", cfg.Kafka.ClientID)
		fmt.Printf("  RetryMax: %d\n", kafkaConfig.Producer.Retry.Max)
		fmt.Printf("  RetryBackoff: %s\n", kafkaConfig.Producer.Retry.Backoff)
		fmt.Printf("  RequiredAcks: %s\n", cfg.Kafka.GetRequiredAcks())
		fmt.Printf("  MaxMessageBytes: %d\n", kafkaConfig.Producer.MaxMessageBytes)
		fmt.Printf("  Timeout: %s\n\n", kafkaConfig.Producer.Timeout)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(cfg.Logger.GetFormatter())
	logger.Info("starting postgres2kafka connector...")

	kafka, err := sarama.NewSyncProducer(cfg.Kafka.Brokers, kafkaConfig)
	if err != nil {
		return err
	}
	defer kafka.Close()

	conn, err := pgconn.Connect(ctx, cfg.Postgres.ConnectionURI)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if !flags.dropSlot && flags.renewSlot {
		logger.Info("RENEW SLOT")
		if err := postgres.DropSlot(ctx, conn, cfg.Postgres.GetSlotName()); err != nil {
			return err
		}
		return postgres.CreateSlot(ctx, conn, cfg.Postgres.GetSlotName())
	}

	if flags.dropSlot {
		logger.Info("DROP SLOT")
		return postgres.DropSlot(ctx, conn, cfg.Postgres.GetSlotName())
	}

	if err := postgres.CreateSlot(ctx, conn, cfg.Postgres.GetSlotName()); err != nil {
		return err
	}

	var published uint64
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				val := atomic.LoadUint64(&published)
				logger.WithField("published", val).Infof("kafka")
			case <-ctx.Done():
				return
			}
		}
	}()

	walListener := postgres.NewWalReceiverWithContext(ctx, conn, cfg.Postgres.SlotName)
	walListener.OnError(func(err *pgproto3.ErrorResponse) {
		logger.Error(err)
	})
	walListener.OnStandbyStatusUpdate(func(lsn pglogrepl.LSN) {
		logger.
			WithField("clientLSN", lsn).
			Infof("postgres standby status update")
	})
	walListener.OnPrimaryKeepalive(func(message pglogrepl.PrimaryKeepaliveMessage) {
		logger.
			WithField("serverLSN", message.ServerWALEnd).
			WithField("serverTime", message.ServerTime).
			WithField("replyRequested", message.ReplyRequested).
			Infof("postgres primary keepalive")
	})
	walListener.OnExit(func() {
		logger.Info("closed")
	})
	walListener.OnMessage(
		func(streamName string, streamID string, _ int64, rawData []byte) error {
			if len(streamName) == 0 && len(streamID) == 0 || len(rawData) == 0 {
				logger.Warn("INVALID DATA SCHEME gulfstream.outbox")
				return nil
			}
			_, _, err := kafka.SendMessage(&sarama.ProducerMessage{
				Topic: streamName,
				Key:   sarama.StringEncoder(streamName + streamID),
				Value: sarama.ByteEncoder(rawData),
			})
			if err == nil {
				atomic.AddUint64(&published, 1)
			}
			return err
		})
	return walListener.Listen()
}
