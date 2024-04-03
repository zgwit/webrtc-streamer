package signaling

import "github.com/gorilla/websocket"

//call url：options:

type Viewer struct {
	ws *websocket.Conn
}

func (v *Viewer) Close() {
	_ = v.ws.Close()
}
