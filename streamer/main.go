package main

import (
	"time"
	_ "webrtc-streamer/camera" //默认支持RTSP协议
	_ "webrtc-streamer/rtsp"   //默认支持RTSP协议

	"github.com/busy-cloud/boat/log"
)

func main() {

	err := Load()
	if err != nil {
		log.Fatal(err)
	}

	//持续连接服务器
	for {
		err = Open()
		if err != nil {
			log.Error(err)
		}
		time.Sleep(time.Second)
	}
}
