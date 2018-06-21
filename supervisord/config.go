package supervisord

import (
	"github.com/BurntSushi/toml"
)

func newSupervisordConfig() *SupervisorConfig {
	return &SupervisorConfig{}
}

func ParseConfigFile(filepath string) (*SupervisorConfig, error) {
	cfg := newSupervisordConfig()
	_, err := toml.DecodeFile(filepath, cfg)
	return cfg, err
}

func ParseConfigString(data string) (*SupervisorConfig, error) {
	cfg := newSupervisordConfig()
	_, err := toml.Decode(data, cfg)
	return cfg, err
}
