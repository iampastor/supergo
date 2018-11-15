package supervisord

import (
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type SupervisorConfig struct {
	Include struct {
		Files string `toml:"files"`
	} `toml:"include"`
	ProgramConfigs map[string]*ProgramConfig `toml:"program"`
}

type ProgramConfig struct {
	Directory         string   `toml:"directory" json:"directory"`
	Command           string   `toml:"command" json:"command"`
	Args              []string `toml:"args" json:"args"`
	AutoRestart       bool     `toml:"auto_restart" json:"auto_restart"`
	StdoutLogFile     string   `toml:"stdout_logfile" json:"stdout_logfile"`
	StderrLogFile     string   `toml:"stderr_logfile" json:"stderr_logfile"`
	MaxRetry          int      `toml:"max_retry" json:"max_retry"`
	ListenAddrs       []string `toml:"listen_addrs" json:"listen_addrs"`
	StopTimeout       int      `toml:"stop_timeout" json:"stop_timeout"`
	StopBeforeRestart bool     `toml:"stop_before_restart" json:"stop_before_restart"`
}

func newSupervisordConfig() *SupervisorConfig {
	return &SupervisorConfig{
		ProgramConfigs: make(map[string]*ProgramConfig),
	}
}

func ParseConfigFile(filepath string) (*SupervisorConfig, error) {
	cfg := newSupervisordConfig()
	_, err := toml.DecodeFile(filepath, cfg)
	if err != nil {
		return nil, err
	}
	files := getConfigFiles(cfg.Include.Files)
	for _, file := range files {
		subCfg, err := parseConfig(file)
		if err != nil {
			log.Printf("parse config file %s %s", file, err.Error())
			continue
		}
		for name, c := range subCfg.ProgramConfigs {
			if c.StopTimeout == 0 {
				c.StopTimeout = 10
			}
			if c.MaxRetry == 0 {
				c.MaxRetry = 3
			}
			cfg.ProgramConfigs[name] = c
		}
	}

	return cfg, nil
}

func getConfigFiles(configPath string) []string {
	var files []string

	if configPath == "" {
		return files
	}

	configDir, namepatten := path.Split(configPath)
	filepath.Walk(configDir, func(cpath string, info os.FileInfo, err error) error {
		_, fname := path.Split(cpath)
		if err != nil {
			log.Printf("parse config file files %s %s", cpath, err.Error())
			return nil
		}
		if ok, _ := filepath.Match(namepatten, fname); ok {
			files = append(files, cpath)
		}
		return nil
	})

	return files
}

func parseConfig(filename string) (*SupervisorConfig, error) {
	cfg := newSupervisordConfig()
	_, err := toml.DecodeFile(filename, cfg)
	return cfg, err
}
