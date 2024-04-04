package signaling

import (
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/lib"
	"github.com/zgwit/iot-master/v4/pkg/log"
)

type Server struct {
	workers lib.Map[Worker]
}

// ConnectWorker 注册Worker
func (s *Server) ConnectWorker(id string, ws *websocket.Conn) {
	worker := s.workers.Load(id)
	if worker != nil {
		worker.Close()
	}

	worker = &Worker{id: id, ws: ws}
	s.workers.Store(id, worker)

	worker.Serve()
}

func (s *Server) ConnectViewer(id string, ws *websocket.Conn) {
	worker := s.workers.Load(id)
	if worker == nil {
		log.Errorf("worker %s not exits ", id)
		return
	}
	worker.Connect(ws)
}
