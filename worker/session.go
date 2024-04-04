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

func (s *Session) Close() error {
	return s.pc.Close()
}

func (s *Session) Report(tp string, data string) {
	msg := signaling.Message{Id: s.Id, Type: tp, Data: data}
	err := WriteMessage(&msg)
	if err != nil {
		log.Error(err)
	}
}

func (s *Session) Handle(msg *signaling.Message) {
	switch msg.Type {
	case "ice":

	case "connect":
		s.handleConnect(msg.Data)
	case "disconnect":
		s.handleDisconnect(msg.Data)
	case "offer":
		s.handleOffer(msg.Data)
	case "answer":
		s.handleAnswer(msg.Data)
	case "candidate":
		s.handleCandidate(msg.Data)
	}
}

func (s *Session) handleConnect(data string) {
	var arg signaling.Connect
	err := json.Unmarshal([]byte(data), &arg)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	s.source, err = source.Get(arg.Url, arg.Options)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	err = s.createPeerConnection()
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	//Attach
	err = s.source.Attach(s.Id, s.pc)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	err = s.pc.SetLocalDescription(offer)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	//等待ice收集完成
	//gc := webrtc.GatheringCompletePromise(pc)

	//s.Report("offer", offer.SDP)
	s.Report("offer", s.pc.LocalDescription().SDP)
}

func (s *Session) handleDisconnect(data string) {
	sessions.Delete(s.Id)
}

func (s *Session) handleOffer(sdp string) {

	err := s.createPeerConnection()
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	if err = s.pc.SetRemoteDescription(offer); err != nil {
		s.Report("error", err.Error())
		return
	}

	//Attach
	err = s.source.Attach(s.Id, s.pc)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	//回复消息
	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	//等待ice收集完成
	//gc := webrtc.GatheringCompletePromise(pc)

	if err = s.pc.SetLocalDescription(answer); err != nil {
		s.Report("error", err.Error())
		return
	}
	//<-gc

	//将sdp发送到浏览器
	//s.Report("answer", answer.SDP)
	s.Report("answer", s.pc.LocalDescription().SDP)
}

func (s *Session) handleCandidate(str string) {
	var candidate webrtc.ICECandidateInit
	err := json.Unmarshal([]byte(str), &candidate)
	if err != nil {
		s.Report("error", err.Error())
		return
	}
	err = s.pc.AddICECandidate(candidate)
	if err != nil {
		s.Report("error", err.Error())
		return
	}
}

func (s *Session) createPeerConnection() (err error) {
	config := webrtc.Configuration{SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback}

	if len(config.ICEServers) == 0 {
		config.ICEServers = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}
	}

	me := &webrtc.MediaEngine{}
	err = me.RegisterDefaultCodecs()
	if err != nil {
		return
	}

	r := &interceptor.Registry{}
	err = webrtc.RegisterDefaultInterceptors(me, r)
	if err != nil {
		return
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithInterceptorRegistry(r))
	s.pc, err = api.NewPeerConnection(config)
	if err != nil {
		return
	}

	//监听主要事件
	s.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			_ = s.Close()
		}
		s.Report("state", connectionState.String())
	})
	s.pc.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			//_ = d.Report(msg.Data)
		})
	})
	s.pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		can := candidate.ToJSON()
		str, _ := json.Marshal(can)
		//将sdp发送到浏览器
		s.Report("candidate", string(str))
	})

	return
}

func (s *Session) handleAnswer(sdp string) {
	answer := webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}
	err := s.pc.SetRemoteDescription(answer)
	if err != nil {
		s.Report("error", err.Error())
	}
}
