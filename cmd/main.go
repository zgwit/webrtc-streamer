package main

import (
	_ "github.com/zgwit/webrtc-streamer/source"
	"github.com/zgwit/webrtc-streamer/worker"
	"log"
)

func main() {
	err := worker.Open()
	if err != nil {
		log.Fatal(err)
	}
}
