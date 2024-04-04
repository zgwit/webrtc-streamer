package worker

import (
	"encoding/json"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"github.com/zgwit/iot-master/v4/pkg/log"
	"github.com/zgwit/webrtc-streamer/signaling"
	"github.com/zgwit/webrtc-streamer/source"
)

type Session struct {
	Id string

	//streams map[int8]*Stream
	pc *webrtc.PeerConnection

	source source.Source
}

func newSession(id string) *Session {
	m := &Session{Id: id}
	return m
}

func (c *Session) Report(tp string, data string) {
	msg := signaling.Message{Id: c.Id, Type: tp, Data: data}
	err := server.WriteJSON(&msg)
	if err != nil {
		log.Error(err)
	}
}

func (c *Session) Handle(msg *signaling.Message) {
	switch msg.Type {
	case "ice":

	case "connect":
		c.handleConnect(msg.Data)
	case "offer":
		c.handleOffer(msg.Data)
	case "answer":
		c.handleAnswer(msg.Data)
	case "candidate":
		c.handleCandidate(msg.Data)

	}
}

type connectArgs struct {
	Url     string         `json:"url"`
	Options map[string]any `json:"options,omitempty"`
}

func (c *Session) handleConnect(data string) {
	var arg connectArgs
	err := json.Unmarshal([]byte(data), &arg)
	if err != nil {
		c.Report("error", err.Error())
	}

	c.source, err = source.Get(arg.Url, arg.Options)
	if err != nil {
		c.Report("error", err.Error())
	}
}

func (c *Session) handleOffer(sdp string) {
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}

	pc, err := c.NewPeerConnection(webrtc.Configuration{SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback})
	if err != nil {
		c.Report("error", err.Error())
		return
	}

	//AddTracks
	err = c.source.AddTracks(c.Id, pc)
	if err != nil {
		c.Report("error", err.Error())
		return
	}

	//监听主要事件
	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			//_ = c.Close()
		}
		c.Report("state", connectionState.String())
	})
	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			//_ = d.Report(msg.Data)
		})
	})
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		can := candidate.ToJSON()
		str, _ := json.Marshal(can)
		//将sdp发送到浏览器
		c.Report("candidate", string(str))
	})

	if err = pc.SetRemoteDescription(offer); err != nil {
		c.Report("error", err.Error())
		return
	}

	//回复消息
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		c.Report("error", err.Error())
		return
	}

	//等待ice收集完成
	//gc := webrtc.GatheringCompletePromise(pc)

	if err = pc.SetLocalDescription(answer); err != nil {
		c.Report("error", err.Error())
		return
	}

	//<-gc

	//将sdp发送到浏览器
	c.Report("answer", pc.LocalDescription().SDP)

	c.pc = pc
}

func (c *Session) handleCandidate(str string) {
	var candidate webrtc.ICECandidateInit
	err := json.Unmarshal([]byte(str), &candidate)
	if err != nil {
		c.Report("error", err.Error())
		return
	}
	err = c.pc.AddICECandidate(candidate)
	if err != nil {
		c.Report("error", err.Error())
		return
	}
}

func (c *Session) NewPeerConnection(configuration webrtc.Configuration) (*webrtc.PeerConnection, error) {
	if len(configuration.ICEServers) == 0 {
		configuration.ICEServers = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}
	}

	me := &webrtc.MediaEngine{}
	if err := me.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	r := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(me, r); err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithInterceptorRegistry(r))
	return api.NewPeerConnection(configuration)
}

func (c *Session) handleAnswer(sdp string) {
	answer := webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}
	err := c.pc.SetRemoteDescription(answer)
	if err != nil {
		c.Report("error", err.Error())
	}
}
