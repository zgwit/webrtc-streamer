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
	err := server.WriteJSON(&msg)
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
	}

	s.source, err = source.Get(arg.Url, arg.Options)
	if err != nil {
		s.Report("error", err.Error())
	}
}

func (s *Session) handleDisconnect(data string) {
	sessions.Delete(s.Id)
}

func (s *Session) handleOffer(sdp string) {
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}

	pc, err := s.newPeerConnection(webrtc.Configuration{SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback})
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	//AddTracks
	err = s.source.AddTracks(s.Id, pc)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	//监听主要事件
	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			_ = s.Close()
		}
		s.Report("state", connectionState.String())
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
		s.Report("candidate", string(str))
	})

	if err = pc.SetRemoteDescription(offer); err != nil {
		s.Report("error", err.Error())
		return
	}

	//回复消息
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		s.Report("error", err.Error())
		return
	}

	//等待ice收集完成
	//gc := webrtc.GatheringCompletePromise(pc)

	if err = pc.SetLocalDescription(answer); err != nil {
		s.Report("error", err.Error())
		return
	}

	//<-gc

	//将sdp发送到浏览器
	s.Report("answer", pc.LocalDescription().SDP)

	s.pc = pc
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

func (s *Session) newPeerConnection(configuration webrtc.Configuration) (*webrtc.PeerConnection, error) {
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

func (s *Session) handleAnswer(sdp string) {
	answer := webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}
	err := s.pc.SetRemoteDescription(answer)
	if err != nil {
		s.Report("error", err.Error())
	}
}
