package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port uint16 `json:"port"`
}

var config = Config{
	Port: 8080,
}

const cfg = "server.yaml"

func Load() error {
	buf, err := os.ReadFile(cfg)
	if err != nil {
		_ = Store()
		return err
	}
	return yaml.Unmarshal(buf, &config)
}

func Store() error {
	buf, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	return os.WriteFile(cfg, buf, os.ModePerm)
}
