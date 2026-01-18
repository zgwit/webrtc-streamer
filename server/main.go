package main

import (
	"fmt"
	"net/http"
	"webrtc-streamer/signaling"

	"github.com/busy-cloud/boat/log"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
	err := Load()
	if err != nil {
		log.Fatal(err)
	}

	//创建服务器
	app := gin.Default()
	app.Use(cors.Default())

	//推流接入
	app.GET("stream/:id", func(ctx *gin.Context) {
		ws, err := upper.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Error(err)
			return
		}

		//注册
		server.ConnectStreamer(ctx.Param("id"), ws)
	})

	//客户端
	app.GET("view/:id", func(ctx *gin.Context) {
		ws, err := upper.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Error(err)
			return
		}

		server.ConnectViewer(ctx.Param("id"), ws)
	})

	//开始监听
	addr := fmt.Sprintf(":%d", config.Port)
	err = app.Run(addr)
	if err != nil {
		log.Error(err)
		return
	}

	//TODO TLS
}
