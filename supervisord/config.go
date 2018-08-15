package supervisord

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type SupervisorConfig struct {
	Include struct {
		Files string `toml:"files"`
	} `toml:"include"`
	ProgramConfigs map[string]*ProgramConfig `toml:"program"`
}

type ProgramConfig struct {
	Directory   string   `toml:"directory" json:"directory"`
	Command     string   `toml:"command" json:"command"`
	Args        []string `toml:"args" json:"args"`
	AutoRestart bool     `toml:"auto_restart" json:"auto_restart"`
	StdoutFile  string   `toml:"stdout_file" json:"stdout_file"`
	StderrFile  string   `toml:"stderr_file" json:"stderr_file"`
	MaxRetry    int      `toml:"max_retry" json:"max_retry"`
	ListenAddrs []string `toml:"listen_addrs" json:"listen_addrs"`
	StopTimeout int      `toml:"stop_timeout" json:"stop_timeout"`
}

func newSupervisordConfig() *SupervisorConfig {
	return &SupervisorConfig{}
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
			cfg.ProgramConfigs[name] = c
		}
	}

	return cfg, nil
}

func getConfigFiles(configPath string) []string {
	var files []string

	configDir, filename := path.Split(configPath)
	filename = strings.Replace(filename, "*", ".*", -1)
	re, err := regexp.Compile(filename)
	if err != nil {
		log.Printf("parse config file files %s", err.Error())
		return files
	}
	filepath.Walk(configDir, func(cpath string, info os.FileInfo, err error) error {
		_, filename := path.Split(cpath)
		if err != nil {
			log.Printf("parse config file files %s %s", cpath, err.Error())
			return nil
		}
		if re.MatchString(filename) {
			files = append(files, path.Join(configDir, filename))
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
