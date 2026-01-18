package main

import (
	"os"

	"github.com/busy-cloud/boat/lib"
	"github.com/pion/webrtc/v4"
	"gopkg.in/yaml.v3"
)

type Config struct {
	IceServers []webrtc.ICEServer `json:"ice_servers"`
	Server     string             `json:"server"`
	Id         string             `json:"id"`
}

var config = Config{
	IceServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	Server:     "ws://localhost:8080",
}

const cfg = "streamer.yaml"

func Load() error {
	buf, err := os.ReadFile(cfg)
	if err != nil {
		config.Id = lib.RandomString(16) //生成默认ID
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
