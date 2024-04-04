package main

import (
	"github.com/spf13/viper"
	"github.com/zgwit/iot-master/v4/pkg/config"
	_ "github.com/zgwit/webrtc-streamer/source"
	"github.com/zgwit/webrtc-streamer/worker"
	"log"
)

func main() {
	config.Name("webrtc-streamer")
	viper.SetDefault("server", "ws://localhost:8080/worker/test")

	err := config.Load()
	if err != nil {
		_ = config.Store()
	}

	err = worker.Open(viper.GetString("server"))
	if err != nil {
		log.Fatal(err)
	}
}
