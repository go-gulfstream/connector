package config

type kafka struct {
	Brokers []string `yaml:"brokers" mapstructure:"BROKERS"`
}

func (k kafka) Validate() error {
	return nil
}
