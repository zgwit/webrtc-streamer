package camera

import (
	"errors"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/prop"
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

	c.stream, err = mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			constraint.Width = prop.Int(640)
			constraint.Height = prop.Int(480)
			//TODO 参数 宽高，帧率，编码，等
		},
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
