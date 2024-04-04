package worker

import (
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/lib"
	"github.com/zgwit/iot-master/v4/pkg/log"
	"github.com/zgwit/webrtc-streamer/signaling"
)

var server *websocket.Conn

var sessions lib.Map[Session]

func Open(url string) (err error) {
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
