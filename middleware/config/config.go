package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	URL string `yaml:"url"`
}

type GatewayConfig struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

type Config struct {
	Gateway  GatewayConfig            `yaml:"gateway"`
	Services map[string]ServiceConfig `yaml:"services"`

	AdminUser string `env:"ADMIN_USER"`
	AdminPass string `env:"ADMIN_PASSWORD"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err = yaml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}
	if err = cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
