package signaling

import (
	"github.com/gorilla/websocket"
	"log"
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
	clients map[string]*Client
}

func (s *Device) transport() {
	for {
		var msg Message
		err := s.ws.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			break
		}
		//转发
		if msg.Client != "" {
			if c, ok := s.clients[msg.Client]; ok {
				err := c.ws.WriteJSON(msg)
				if err != nil {
					_ = c.ws.Close()
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
	devices map[string]*Device
}

func (s *Server) Register(id string, ws *websocket.Conn) {
	if s.devices == nil {
		s.devices = make(map[string]*Device)
	}
	if dev, ok := s.devices[id]; ok {
		dev.ws = ws
		go dev.transport()
	} else {
		s.devices[id] = &Device{ws: ws}
	}
}

func (s *Server) Connect(id string, ws *websocket.Conn) {
	if s.devices == nil {
		s.devices = make(map[string]*Device)
	}
	s.devices[id] = &Device{ws: ws}
}
