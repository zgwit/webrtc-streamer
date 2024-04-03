package worker

import (
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/lib"
	"github.com/zgwit/iot-master/v4/pkg/config"
	"github.com/zgwit/iot-master/v4/pkg/log"
	"github.com/zgwit/webrtc-streamer/signaling"
)

var server *websocket.Conn
var clients lib.Map[Client]

func Open() (err error) {
	//server, err = websocket.Dial("", "ws", "")
	server, _, err = websocket.DefaultDialer.Dial(config.GetString(MODULE, "url"), nil)
	if err != nil {
		return err
	}
	//TODO 守护进程

	go receive()
	return
}

func receive() {
	for {
		var msg signaling.Message
		err := server.ReadJSON(&msg)
		if err != nil {
			log.Error(err)
			break
		}

		client := clients.Load(msg.Id)
		if client == nil {
			client = NewClient(nil)
			clients.Store(msg.Id, client)
		}

		client.Handle(&msg)
	}
}
