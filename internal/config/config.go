package config

type Validator interface {
	Validate() error
}

type Config struct {
	Kafka    kafka    `mapstructure:"kafka"`
	Logger   logger   `mapstructure:"logger"`
	Nats     nats     `mapstructure:"nats"`
	Postgres postgres `mapstructure:"postgres"`
	Redis    redis    `mapstructure:"redis"`
}

func Validate(validators ...Validator) (err error) {
	for _, v := range validators {
		if err = v.Validate(); err != nil {
			return err
		}
	}
	return
}
