package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zgwit/iot-master/v4/log"
	"github.com/zgwit/webrtc-streamer/signaling"
	"net/http"
)

var upper = &websocket.Upgrader{
	//HandshakeTimeout: time.Second,
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	Subprotocols:    []string{"webrtc"},
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var server signaling.Server

func main() {
	app := gin.Default()
	app.Use(cors.Default())

	app.GET("streamer/:id", func(ctx *gin.Context) {
		ws, err := upper.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Error(err)
			return
		}

		//注册
		server.ConnectStreamer(ctx.Param("id"), ws)
	})

	app.GET("streamer/:id/connect", func(ctx *gin.Context) {
		ws, err := upper.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Error(err)
			return
		}

		server.ConnectViewer(ctx.Param("id"), ws)
	})

	err := app.Run(":8080")
	if err != nil {
		log.Error(err)
		return
	}
}
