package signaling

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/lib"
)

type Server struct {
	workers lib.Map[Worker]
}

// Register 注册Worker
func (s *Server) Register(id string, ws *websocket.Conn) {
	worker := s.workers.Load(id)
	if worker != nil {
		worker.ws = ws
		go worker.receive()
	} else {
		worker = &Worker{ws: ws}
		s.workers.Store(id, worker)
	}
}

func (s *Server) Connect(id string, ws *websocket.Conn) error {
	worker := s.workers.Load(id)
	if worker == nil {
		return fmt.Errorf("worker not exits ", id)
	}
	return worker.Connect(ws)
}
