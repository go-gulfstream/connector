package commands

import (
	"os"
	"strings"

	"github.com/go-gulfstream/connector/internal/config"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

func New() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "gc-connector",
		Short: "Exporting gulfstream-events from stream storage to eventbus",
	}

	var configFile string
	root.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is $PWD/gc-connector.yaml)")
	cfg, err := parseConfig(configFile)
	if err != nil {
		return nil, err
	}

	postgres2kafka := postgres2kafkaCommand(cfg)
	postgres2nats := postgres2natsCommand(cfg)
	redis2kafka := redis2kafkaCommand(cfg)
	redis2nats := redis2natsCommand(cfg)

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case postgres2kafka.Name():
			return config.Validate(cfg.Postgres, cfg.Kafka)
		case postgres2nats.Name():
			return config.Validate(cfg.Postgres, cfg.Nats)
		case redis2kafka.Name():
			return config.Validate(cfg.Redis, cfg.Kafka)
		case redis2nats.Name():
			return config.Validate(cfg.Redis, cfg.Nats)
		}
		return nil
	}

	root.AddCommand(postgres2kafka)
	root.AddCommand(postgres2nats)
	root.AddCommand(redis2kafka)
	root.AddCommand(redis2nats)

	return root, nil
}

func parseConfig(configFile string) (*config.Config, error) {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// default path.
		viper.AddConfigPath("./")
		viper.SetConfigType("yml")
		viper.SetConfigName("gc-connector")
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// overwrite config from environment variables if exists.
	for _, k := range viper.AllKeys() {
		envKey := "GS_" + strings.Replace(k, ".", "_", -1)
		envKey = strings.ToUpper(envKey)
		if envVal := os.Getenv(envKey); len(envVal) > 0 {
			viper.Set(k, envVal)
		}
	}

	cfg := new(config.Config)
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
