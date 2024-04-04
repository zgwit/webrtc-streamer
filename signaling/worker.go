package signaling

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/lib"
	"github.com/zgwit/iot-master/v4/pkg/log"
)

type Worker struct {
	id      string
	ws      *websocket.Conn
	viewers lib.Map[Viewer] //viewers map[string]*Viewer
}

func (w *Worker) Close() {
	_ = w.ws.Close()
	w.viewers.Range(func(_ string, v *Viewer) bool {
		v.Close()
		return true
	})
}

func (w *Worker) Serve() {
	for {
		var msg Message
		err := w.ws.ReadJSON(&msg)
		if err != nil {
			log.Error(err)
			break
		}
		log.Trace("worker receive", w.id, msg)

		//转发
		if msg.Id != "" {
			client := w.viewers.Load(msg.Id)
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

func (w *Worker) Connect(ws *websocket.Conn) {
	cid := uuid.NewString()
	log.Info("connect ", cid)

	viewer := &Viewer{ws: ws}

	w.viewers.Store(cid, viewer)

	//通知连接
	//_ = w.ws.WriteJSON(&Message{Id: cid, Type: "connect"})

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Error(err)
			break
		}
		log.Trace("viewer receive", w.id, cid, msg)

		msg.Id = cid
		err = w.ws.WriteJSON(msg)
		if err != nil {
			log.Error(err)
			break
		}
	}
	log.Info("finished ", cid)

	return
}
