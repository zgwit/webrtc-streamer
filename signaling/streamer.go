package signaling

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/lib"
	"github.com/zgwit/iot-master/v4/log"
	"sync"
)

type Streamer struct {
	id      string
	ws      *websocket.Conn
	wsLock  sync.Mutex
	viewers lib.Map[Viewer] //viewers map[string]*Viewer
}

func (s *Streamer) Close() {
	_ = s.ws.Close()
	s.viewers.Range(func(_ string, v *Viewer) bool {
		v.Close()
		return true
	})
}

func (s *Streamer) Serve() {
	for {
		var msg Message
		err := s.ws.ReadJSON(&msg)
		if err != nil {
			log.Error(err)
			break
		}
		log.Trace("streamer receive", s.id, msg)

		//转发
		if msg.Id != "" {
			client := s.viewers.Load(msg.Id)
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

func (s *Streamer) WriteMessage(msg *Message) error {
	s.wsLock.Lock()
	defer s.wsLock.Unlock()

	return s.ws.WriteJSON(msg)
}

func (s *Streamer) Connect(ws *websocket.Conn) {
	cid := uuid.NewString()
	log.Info("connect ", cid)

	viewer := &Viewer{ws: ws}

	s.viewers.Store(cid, viewer)

	//通知连接
	//_ = s.ws.WriteJSON(&Message{Id: cid, Type: "connect"})

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Error(err)
			break
		}
		log.Trace("viewer receive", s.id, cid, msg)

		msg.Id = cid
		err = s.WriteMessage(&msg)
		if err != nil {
			log.Error(err)
			break
		}
	}
	log.Info("finished ", cid)

	return
}
