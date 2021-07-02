package config

type postgres struct {
	SlotName string `yaml:"slotName"`
}

func (p postgres) Validate() error {
	return nil
}
