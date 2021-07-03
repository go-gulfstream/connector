package config

import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
)

type kafka struct {
	Brokers         []string      `mapstructure:"brokers"`
	ClientID        string        `mapstructure:"clientID"`
	RetryMax        int           `mapstructure:"retryMax"`
	RetryBackoff    time.Duration `mapstructure:"retryBackoff"`
	RequiredAcks    string        `mapstructure:"requiredAcks"`
	MaxMessageBytes int           `mapstructure:"maxMessageBytes"`
	Timeout         time.Duration `mapstructure:"timeout"`
}

func (k kafka) Validate() error {
	if len(k.Brokers) == 0 {
		return fmt.Errorf("config: kafka broker list is empty")
	}
	return nil
}

func (k kafka) NewConfig() *sarama.Config {
	conf := sarama.NewConfig()
	if len(k.ClientID) == 0 {
		conf.ClientID = "postgres2kafka"
	}

	conf.Producer.Return.Successes = true
	conf.Producer.Return.Errors = true
	if k.RetryMax > 0 {
		conf.Producer.Retry.Max = k.RetryMax
	}
	if k.RetryBackoff > 0 {
		conf.Producer.Retry.Backoff = k.RetryBackoff
	}
	switch k.RequiredAcks {
	case "local":
		conf.Producer.RequiredAcks = sarama.WaitForLocal
	default:
		conf.Producer.RequiredAcks = sarama.WaitForAll
	}
	if k.MaxMessageBytes > 0 {
		conf.Producer.MaxMessageBytes = k.MaxMessageBytes
	}
	if k.Timeout > 0 {
		conf.Producer.Timeout = k.Timeout
	}
	return conf
}

func (k kafka) GetRequiredAcks() string {
	if k.RequiredAcks == "local" {
		return "local"
	}
	return "waitForAll"
}
