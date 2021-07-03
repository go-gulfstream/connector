package config

import (
	"fmt"
	"strings"
)

const defaultSlotName = "gulstream"

type postgres struct {
	ConnectionURI string `mapstructure:"connectionURI"`
	SlotName      string `mapstructure:"slotName"`
}

func (p postgres) Validate() error {
	if len(p.ConnectionURI) == 0 {
		return fmt.Errorf("config: postgres connection URI is empty")
	}
	if !strings.Contains(p.ConnectionURI, "?replication=database") {
		return fmt.Errorf("config: postgres replication mode is disabled. turn on [?replication=database]")
	}
	return nil
}

func (p postgres) GetSlotName() string {
	if len(p.SlotName) == 0 {
		return defaultSlotName
	}
	return p.SlotName
}
