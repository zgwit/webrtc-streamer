package signaling

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/lib"
	"github.com/zgwit/iot-master/v4/pkg/log"
)

type Message struct {
	Client string
	Type   string
	Data   string
}

//call url：options:

type Client struct {
	ws *websocket.Conn
}

type Device struct {
	ws      *websocket.Conn
	clients lib.Map[Client] //clients map[string]*Client
}

func (s *Device) transport() {
	for {
		var msg Message
		err := s.ws.ReadJSON(&msg)
		if err != nil {
			log.Error(err)
			break
		}
		//转发
		if msg.Client != "" {
			client := s.clients.Load(msg.Client)
			if client != nil {
				err = client.ws.WriteJSON(msg)
				if err != nil {
					log.Error(err)
					_ = client.ws.Close()
				}
			}
			continue
		}

		//TODO 处理非桥接数据
		//switch msg.Type {
		//case "":
		//}
	}

}

type Server struct {
	devices lib.Map[Device]
}

func (s *Server) Register(id string, ws *websocket.Conn) {
	dev := s.devices.Load(id)
	if dev != nil {
		dev.ws = ws
		go dev.transport()
	} else {
		dev = &Device{ws: ws}
		s.devices.Store(id, dev)
	}
}

func (s *Server) Connect(id string, ws *websocket.Conn) {
	dev := s.devices.Load(id)
	if dev == nil {
		log.Error("device not exits ", id)
		return
	}

	cid := uuid.NewString()
	log.Info("connect ", id, cid)

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Error(err)
			break
		}

		msg.Client = cid
		err = dev.ws.WriteJSON(msg)
		if err != nil {
			log.Error(err)
			break
		}
	}
	log.Info("finished ", id, cid)
}
