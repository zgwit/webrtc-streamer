package rtsp

import (
	"errors"
	"github.com/deepch/vdk/format/rtspv2"
	"github.com/pion/webrtc/v3"
	"github.com/zgwit/iot-master/v4/lib"
	"time"
)

type Camera struct {
	Url   string
	Audio bool

	//RTSP连接
	rtsp *rtspv2.RTSPClient

	clients lib.Map[Session]
}

func (c *Camera) Check() error {
	if c.rtsp != nil {
		return nil
	}

	var err error
	c.rtsp, err = rtspv2.Dial(rtspv2.RTSPClientOptions{
		URL:              c.Url,
		DialTimeout:      3 * time.Second,
		ReadWriteTimeout: 3 * time.Second,
		DisableAudio:     !c.Audio,
	})
	if err != nil {
		return err
	}

	go c.receive()

	return nil
}

func (c *Camera) Attach(cid string, pc *webrtc.PeerConnection) error {
	if c.rtsp == nil {
		return errors.New("未连接")
	}

	s := newSession(pc)
	c.clients.Store(cid, s)

	return s.attach(c.rtsp.CodecData)
}

func (c *Camera) handleCodecUpdate() {
	//TODO 动态添加到pc上
}

func (c *Camera) receive() {
	defer func() {
		c.rtsp.Close()
		c.rtsp = nil
	}()

	autoClose := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-autoClose.C:
			return
		case signals := <-c.rtsp.Signals:
			switch signals {
			case rtspv2.SignalCodecUpdate:
				c.handleCodecUpdate()
			case rtspv2.SignalStreamRTPStop:
				return
			}
		case pkt := <-c.rtsp.OutgoingPacketQueue:
			if pkt.IsKeyFrame {
				autoClose.Reset(20 * time.Second)
			}
			c.clients.Range(func(_ string, client *Session) bool {
				client.Put(pkt)
				//if len(client.queue) < cap(client.queue) {
				//	client.queue <- pkt
				//}
				return true
			})
		}
	}
}
