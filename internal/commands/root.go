package commands

import (
	"os"
	"strings"

	"github.com/go-gulfstream/connector/internal/config"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

var envMap = map[string]string{
	"postgres.connectionURI": "GS_POSTGRES_CONNECTIONURI",
	"postgres.slotName":      "GS_POSTGRES_SLOTNAME",
	"kafka.brokers":          "GS_KAFKA_BROKERS",
	"kafka.clientID":         "GS_KAFKA_CLIENTID",
	"kafka.retryMax":         "GS_KAFKA_RETRYMAX",
	"kafka.retryBackoff":     "GS_KAFKA_RETRYBACKOFF",
	"kafka.requiredAcks":     "GS_KAFKA_REQUIREDACKS",
	"kafka.maxMessageBytes":  "GS_KAFKA_MAXMESSAGEBYTES",
	"kafka.timeout":          "GS_KAFKA_TIMEOUT",
}

func New() (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "gc-connector",
		Short: "Exporting gulfstream-events from stream storage to eventbus",
	}

	root.AddCommand(postgres2kafkaCommand())

	return root, nil
}

func parseConfig(configFile string) (*config.Config, error) {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// default path.
		viper.AddConfigPath(".")
		viper.SetConfigType("yml")
		viper.SetConfigName("gs-connector")
	}

	if err := viper.ReadInConfig(); err != nil {
		if !isEnabledEnvConfig() {
			return nil, err
		}
	}

	if len(viper.AllKeys()) == 0 {
		// make config from environment variables if exists.
		for k, v := range envMap {
			viper.Set(k, os.Getenv(v))
		}
	} else {
		// overwrite config from environment variables if exists.
		for _, k := range viper.AllKeys() {
			envKey := "GS_" + strings.Replace(k, ".", "_", -1)
			envKey = strings.ToUpper(envKey)
			if envVal := os.Getenv(envKey); len(envVal) > 0 {
				viper.Set(k, envVal)
			}
		}
	}

	cfg := new(config.Config)
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func isEnabledEnvConfig() bool {
	for _, env := range envMap {
		if len(os.Getenv(env)) > 0 {
			return true
		}
	}
	return false
}
