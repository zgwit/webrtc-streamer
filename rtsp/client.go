package rtsp

import (
	"github.com/deepch/vdk/av"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	connection *webrtc.PeerConnection
	streams    map[int]*Stream
	queue      chan *av.Packet
}

func newClient(connection *webrtc.PeerConnection) *Client {
	client := &Client{
		connection: connection,
		streams:    make(map[int]*Stream),
		queue:      make(chan *av.Packet, 100),
	}
	go client.sending()
	return client
}

func (c *Client) Put(pkt *av.Packet) {
	if len(c.queue) < cap(c.queue) {
		c.queue <- pkt
	}
}

func (c *Client) sending() {
	for {
		pkt := <-c.queue
		if pkt == nil {
			break
		}

		stream, ok := c.streams[int(pkt.Idx)]
		if !ok {
			continue
		}

		err := stream.Write(pkt)
		if err != nil {
			break
		}
	}
}
