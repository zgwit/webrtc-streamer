package rtsp

import (
	"bytes"
	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/codec/h264parser"
	"github.com/deepch/vdk/codec/h265parser"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/zgwit/iot-master/v4/log"
)

type Stream struct {
	codec av.CodecData
	track *webrtc.TrackLocalStaticSample
}

func (t *Stream) Write(pkt *av.Packet) error {
	switch t.codec.Type() {
	case av.H264:
		nalus, _ := h264parser.SplitNALUs(pkt.Data)
		for _, nalu := range nalus {
			naltype := nalu[0] & 0x1f
			if naltype == 5 {
				codec := t.codec.(h264parser.CodecData)
				err := t.track.WriteSample(media.Sample{
					Data: append([]byte{0, 0, 0, 1},
						bytes.Join([][]byte{codec.SPS(), codec.PPS(), nalu}, []byte{0, 0, 0, 1})...),
					Duration: pkt.Duration})
				if err != nil {
					return err
				}
			} else {
				err := t.track.WriteSample(media.Sample{
					Data:     append([]byte{0, 0, 0, 1}, nalu...),
					Duration: pkt.Duration})
				if err != nil {
					return err
				}
			}
		}
	case av.H265:
		nalus, _ := h265parser.SplitNALUs(pkt.Data)
		for _, nalu := range nalus {
			naltype := (nalu[0] & 0x7e) >> 1
			if naltype == 5 {
				codec := t.codec.(h265parser.CodecData)
				err := t.track.WriteSample(media.Sample{
					Data: append([]byte{0, 0, 0, 1},
						bytes.Join([][]byte{codec.VPS(), codec.SPS(), codec.PPS(), nalu}, []byte{0, 0, 0, 1})...),
					Duration: pkt.Duration})
				if err != nil {
					return err
				}
			} else {
				err := t.track.WriteSample(media.Sample{
					Data:     append([]byte{0, 0, 0, 1}, nalu...),
					Duration: pkt.Duration})
				if err != nil {
					return err
				}
			}
		}
	case av.PCM_ALAW, av.OPUS, av.PCM_MULAW:
		err := t.track.WriteSample(media.Sample{Data: pkt.Data, Duration: pkt.Duration})
		if err != nil {
			log.Error(err)
			break
		}
	case av.AAC:
		//TODO: NEED ADD DECODER AND ENCODER
	case av.PCM:
		//TODO: NEED ADD ENCODER
	default:
		//TODO:
	}
	return nil
}
