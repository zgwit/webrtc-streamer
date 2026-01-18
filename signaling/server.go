package signaling

import (
	"github.com/busy-cloud/boat/lib"
	"github.com/busy-cloud/boat/log"
	"github.com/gorilla/websocket"
)

type Server struct {
	streamers lib.Map[Streamer]
}

// ConnectStreamer 注册
func (s *Server) ConnectStreamer(id string, ws *websocket.Conn) {
	streamer := s.streamers.Load(id)
	if streamer != nil {
		streamer.Close()
	}

	streamer = &Streamer{id: id, ws: ws}
	s.streamers.Store(id, streamer)

	streamer.Serve()
}

func (s *Server) ConnectViewer(id string, ws *websocket.Conn) {
	streamer := s.streamers.Load(id)
	if streamer == nil {
		log.Errorf("streamer %s not exits ", id)
		return
	}
	streamer.Connect(ws)
}
