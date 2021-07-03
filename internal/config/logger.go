package config

import (
	"github.com/sirupsen/logrus"
)

type logger struct {
	Formatter string `mapstructure:"formatter"`
	Level     string `mapstructure:"level"`
}

func (l logger) GetFormatter() logrus.Formatter {
	switch l.Formatter {
	case "json":
		return &logrus.JSONFormatter{}
	default:
		return &logrus.TextFormatter{}
	}
}
