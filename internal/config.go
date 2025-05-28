package internal

import (
	"log"
	"os"
	"os/exec"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Commands []*Command `yaml:"commands"`
}

type CommandStatus int

const (
	StatusTrying CommandStatus = iota
	StatusOk
	StatusFailed
)

type Command struct {
	Name string `yaml:"name"`
	Exec string `yaml:"exec"`
	Glob string `yaml:"glob"`
	Cwd  string `yaml:"cwd"`

	LogTail string
	Status  CommandStatus

	proc *exec.Cmd
	mu   sync.Mutex
}

func LoadConfig(configPath string) []*Command {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	return cfg.Commands
}
