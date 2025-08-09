package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Env            string `yaml:"env" env-default:"development"`
	VersionPath    string `yaml:"version_path" env-default:""`
	ExecutableName string `yaml:"executable_name" env-default:""`
	TCPServer      `yaml:"tcp_server"`
}

type TCPServer struct {
	Address     string `yaml:"address" env-default:"0.0.0.0"`
	Port        string `yaml:"port" env-default:"8080"`
	Timeout     int    `yaml:"timeout" env-default:"6"`
	IdleTimeout int    `yaml:"idle_timeout" env-default:"60"`
	WorkerCount int    `yaml:"worker_count" env-default:"1"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		// Если не задан, ищем config.yaml в рабочей директории
		wd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cannot get working directory: %v", err)
		}
		configPath = filepath.Join(wd, "configs/config.yaml")
	}

	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("config file not found: %s", configPath)
	}

	var cfg Config

	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("error reading config file: %s", err)
	}
	return &cfg
}
