package main

import (
	"github.com/spf13/viper"
	"github.com/zgwit/iot-master/v4/pkg/config"
	"github.com/zgwit/iot-master/v4/pkg/log"
	_ "github.com/zgwit/webrtc-streamer/rtsp"
	"github.com/zgwit/webrtc-streamer/worker"
	"time"
)

func main() {
	config.Name("webrtc-streamer")
	viper.SetDefault("server", "ws://localhost:8080/worker/test")

	err := config.Load()
	if err != nil {
		_ = config.Store()
	}

	for {
		err = worker.Open(viper.GetString("server"))
		if err != nil {
			log.Error(err)
		}
		time.Sleep(time.Second)
	}
}
