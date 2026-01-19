package camera

import (
	"errors"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/webrtc/v4"

	// 驱动，需要开启CGO
	_ "github.com/pion/mediadevices/pkg/driver/camera"
	//_ "github.com/pion/mediadevices/pkg/driver/microphone"
	_ "github.com/pion/mediadevices/pkg/driver/screen"
)

type Camera struct {
	Audio bool
	//clients

	stream mediadevices.MediaStream
}

func (c *Camera) Check() (err error) {
	if c.stream != nil {
		return nil
	}

	me := webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()

	// Create a new RTCPeerConnection
	//x264Params, err := x264.NewParams()
	//if err != nil {
	//	return err
	//}
	//x264Params.BitRate = 500_000 // 500kbps

	//x264Params, err := openh264.NewParams()
	//if err != nil {
	//	return err
	//}
	//x264Params.BitRate = 500_000 // 500kbps

	vp9params, err := vpx.NewVP9Params()
	if err != nil {
		return err
	}
	vp9params.BitRate = 500_000

	//opusParams, err := opus.NewParams()
	//if err != nil {
	//	return err
	//}

	c.stream, err = mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			//constraint.FrameFormat = prop.FrameFormatOneOf{frame.FormatI420}
			//constraint.Width = prop.Int(640)
			//constraint.Height = prop.Int(480)
			//TODO 参数 宽高，帧率，编码，等
		},
		Codec: mediadevices.NewCodecSelector(
			mediadevices.WithVideoEncoders(&vp9params),
			//mediadevices.WithAudioEncoders(&opusParams),
		),
	})
	return
}

func (c *Camera) Attach(cid string, pc *webrtc.PeerConnection) error {
	vts := c.stream.GetVideoTracks()
	if len(vts) == 0 {
		return errors.New("没有视频轨")
	}

	_, err := pc.AddTrack(vts[0])
	if err != nil {
		return err
	}

	if c.Audio {
		ats := c.stream.GetAudioTracks()
		if len(ats) > 0 {
			_, err = pc.AddTrack(ats[0])
			if err != nil {
				return err
			}
		}
	}
	return nil
}
