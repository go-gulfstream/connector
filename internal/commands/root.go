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
