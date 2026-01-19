package main

import (
	"sync"
	"webrtc-streamer/signaling"

	"github.com/busy-cloud/boat/lib"
	"github.com/busy-cloud/boat/log"
	"github.com/gorilla/websocket"
)

var server *websocket.Conn

var sessions lib.Map[Session]

func Open() (err error) {
	url := config.Server + "/stream/" + config.Id
	log.Info("connect ", url)

	server, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}

	for {
		var msg signaling.Message
		err = server.ReadJSON(&msg)
		if err != nil {
			break
		}
		log.Println("receive msg ", msg)

		//TODO 删除 session
		s := sessions.Load(msg.Id)
		if s == nil {
			s = newSession(msg.Id)
			sessions.Store(msg.Id, s)
		}

		s.Handle(&msg)
	}
	return
}

var writeLock sync.Mutex

func WriteMessage(msg *signaling.Message) error {
	log.Info("WriteMessage msg ", msg)
	writeLock.Lock()
	defer writeLock.Unlock()
	return server.WriteJSON(&msg)
}
