package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// 全局变量
var (
	converter      *RTSPWebRTCConverter
	converterMutex sync.Mutex
	clients        = make(map[string]*WebRTCClient)
	clientsMutex   sync.RWMutex
)

// WebRTCClient WebRTC客户端
type WebRTCClient struct {
	ID             string
	peerConnection *webrtc.PeerConnection
	videoTrack     *webrtc.TrackLocalStaticRTP
	sendPLITimer   *time.Ticker
}

// RTSPWebRTCConverter RTSP到WebRTC转换器
type RTSPWebRTCConverter struct {
	rtspURL             string
	rtspClient          *gortsplib.Client
	h264Format          *format.H264
	rtpH264Decoder      *rtph264.Decoder
	rtpH264Encoder      *rtph264.Encoder
	isRunning           bool
	startTime           time.Time
	rtspBaseTimestamp   uint32
	webrtcBaseTimestamp uint32
	clockRate           int
	stats               *ConversionStats
}

// ConversionStats 转换统计信息
type ConversionStats struct {
	RTSPPacketsReceived int     `json:"rtspPacketsReceived"`
	WebRTCPacketsSent   int     `json:"webrtcPacketsSent"`
	KeyFramesReceived   int     `json:"keyFramesReceived"`
	ConversionLatency   float64 `json:"conversionLatency"`
	LastRTSPTimestamp   uint32  `json:"lastRtspTimestamp"`
	LastWebRTCTimestamp uint32  `json:"lastWebrtcTimestamp"`
	mutex               sync.RWMutex
}

// NewRTSPWebRTCConverter 创建新的转换器
func NewRTSPWebRTCConverter(rtspURL string) *RTSPWebRTCConverter {
	return &RTSPWebRTCConverter{
		rtspURL:        rtspURL,
		rtpH264Decoder: &rtph264.Decoder{},
		rtpH264Encoder: &rtph264.Encoder{},
		clockRate:      90000, // H264视频标准时钟频率
		stats:          &ConversionStats{},
	}
}

// Start 开始转换
func (c *RTSPWebRTCConverter) Start() error {
	if c.isRunning {
		return fmt.Errorf("converter already running")
	}

	c.isRunning = true
	c.startTime = time.Now()

	go c.connectRTSP()
	return nil
}

// Stop 停止转换
func (c *RTSPWebRTCConverter) Stop() {
	c.isRunning = false
	if c.rtspClient != nil {
		c.rtspClient.Close()
	}
}

// connectRTSP 连接RTSP服务器
func (c *RTSPWebRTCConverter) connectRTSP() {
	for c.isRunning {
		err := c.processRTSPStream()
		if err != nil {
			log.Printf("RTSP stream error: %v, reconnecting in 5 seconds...", err)
			time.Sleep(5 * time.Second)
		}
	}
}

// processRTSPStream 处理RTSP流
func (c *RTSPWebRTCConverter) processRTSPStream() error {
	u, err := base.ParseURL(c.rtspURL)
	if err != nil {
		return fmt.Errorf("failed to parse RTSP URL: %w", err)
	}

	// 创建RTSP客户端
	client := &gortsplib.Client{Scheme: u.Scheme, Host: u.Host}

	// 连接到RTSP服务器
	err = client.Start()
	if err != nil {
		return fmt.Errorf("failed to connect to RTSP server: %w", err)
	}
	defer client.Close()

	c.rtspClient = client

	// 发送OPTIONS请求
	_, err = client.Options(u)
	if err != nil {
		return fmt.Errorf("failed to send OPTIONS request: %w", err)
	}

	// 发送DESCRIBE请求
	sess, _, err := client.Describe(u)
	if err != nil {
		return fmt.Errorf("failed to send DESCRIBE request: %w", err)
	}

	// 解析SDP
	medias := sess.Medias
	if len(medias) == 0 {
		return fmt.Errorf("no media found in SDP")
	}

	// 查找H264视频媒体
	var videoMedia *description.Media
	for _, media := range medias {
		for _, fmt := range media.Formats {
			if h264Fmt, ok := fmt.(*format.H264); ok {
				videoMedia = media
				c.h264Format = h264Fmt

				// 配置H264编码器
				c.rtpH264Encoder.Init()

				log.Printf("Found H264 format - SPS: %d bytes, PPS: %d bytes",
					len(h264Fmt.SPS), len(h264Fmt.PPS))
				break
			}
		}
		if videoMedia != nil {
			break
		}
	}

	if videoMedia == nil {
		return fmt.Errorf("no H264 video format found")
	}

	// 发送SETUP请求
	err = client.SetupAll(u, medias)
	if err != nil {
		return fmt.Errorf("failed to send SETUP request: %w", err)
	}

	// 发送PLAY请求
	_, err = client.Play(nil)
	if err != nil {
		return fmt.Errorf("failed to send PLAY request: %w", err)
	}

	log.Println("RTSP stream started successfully")

	// 处理RTP包
	client.OnPacketRTPAny(func(media *description.Media, f format.Format, packet *rtp.Packet) {
		c.processRTPPacket(packet)
	})

	return err
}

// processRTPPacket 处理RTP包
func (c *RTSPWebRTCConverter) processRTPPacket(rtspPacket *rtp.Packet) {
	startTime := time.Now()

	// 处理第一个RTP包，建立时间戳基准
	if c.rtspBaseTimestamp == 0 {
		c.rtspBaseTimestamp = rtspPacket.Timestamp
		c.webrtcBaseTimestamp = uint32(time.Now().UnixNano() / 1000000)
		log.Printf("Established timestamp base - RTSP: %d, WebRTC: %d",
			c.rtspBaseTimestamp, c.webrtcBaseTimestamp)
	}

	// 解码H264 NALU
	nalus, err := c.rtpH264Decoder.Decode(rtspPacket)
	if err != nil {
		log.Printf("Failed to decode H264 NALU: %v", err)
		return
	}

	// 检查是否包含关键帧
	containsKeyFrame := c.containsKeyFrame(nalus)
	if containsKeyFrame {
		c.stats.mutex.Lock()
		c.stats.KeyFramesReceived++
		c.stats.mutex.Unlock()
		log.Printf("Received key frame at timestamp %d", rtspPacket.Timestamp)
	}

	// 为每个NALU创建WebRTC RTP包
	for _, nalu := range nalus {
		webrtcPackets, err := c.createWebRTCRTPPackets(nalu, rtspPacket)
		if err != nil {
			log.Printf("Failed to create WebRTC RTP packets: %v", err)
			continue
		}

		// 发送到所有WebRTC客户端
		c.sendToWebRTCClients(webrtcPackets)
	}

	// 更新统计信息
	c.stats.mutex.Lock()
	c.stats.RTSPPacketsReceived++
	c.stats.LastRTSPTimestamp = rtspPacket.Timestamp
	c.stats.ConversionLatency = time.Since(startTime).Seconds() * 1000
	c.stats.mutex.Unlock()
}

// containsKeyFrame 检查是否包含关键帧
func (c *RTSPWebRTCConverter) containsKeyFrame(nalus [][]byte) bool {
	for _, nalu := range nalus {
		if len(nalu) > 0 {
			naluType := nalu[0] & 0x1F
			if naluType == 5 { // IDR NALU
				return true
			}
		}
	}
	return false
}

// createWebRTCRTPPackets 创建WebRTC RTP包
func (c *RTSPWebRTCConverter) createWebRTCRTPPackets(nalu []byte, rtspPacket *rtp.Packet) ([]*rtp.Packet, error) {
	var packets []*rtp.Packet

	// 转换时间戳
	webrtcTimestamp := c.convertTimestamp(rtspPacket.Timestamp)

	// 处理不同大小的NALU
	if len(nalu) <= 1400 { // 单包模式
		// 创建WebRTC RTP包
		webrtcPacket := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				Marker:         rtspPacket.Marker,
				PayloadType:    96, // WebRTC H264 payload type
				SequenceNumber: rtspPacket.SequenceNumber,
				Timestamp:      webrtcTimestamp,
				SSRC:           rtspPacket.SSRC,
			},
			Payload: make([]byte, len(nalu)),
		}

		// 复制NALU数据
		copy(webrtcPacket.Payload, nalu)
		packets = append(packets, webrtcPacket)

	} else {
		// 分片模式 - 使用FU-A
		err := c.fragmentAndCreatePackets(nalu, rtspPacket, webrtcTimestamp, &packets)
		if err != nil {
			return nil, err
		}
	}

	return packets, nil
}

// fragmentAndCreatePackets 分片NALU并创建RTP包
func (c *RTSPWebRTCConverter) fragmentAndCreatePackets(nalu []byte, rtspPacket *rtp.Packet, webrtcTimestamp uint32, packets *[]*rtp.Packet) error {
	naluType := nalu[0] & 0x1F
	naluRefIDCBit := nalu[0] & 0x60

	// 分片大小 (MTU - RTP头 - FU-A头)
	fragmentSize := 1400 - 12 - 2

	// 分片数量
	numFragments := (len(nalu) - 1 + fragmentSize - 1) / fragmentSize

	for i := 0; i < numFragments; i++ {
		start := 1 + i*fragmentSize
		end := start + fragmentSize
		if end > len(nalu) {
			end = len(nalu)
		}

		// 创建FU-A头
		fuHeader := []byte{0, 0}
		fuHeader[0] = (naluType & 0x1F) | 0x80 // FU indicator
		fuHeader[1] = naluRefIDCBit | 0x1C     // FU header

		// 设置开始和结束标志
		if i == 0 {
			fuHeader[1] |= 0x80 // 开始标志
		}
		if i == numFragments-1 {
			fuHeader[1] |= 0x40 // 结束标志
		}

		// 创建WebRTC RTP包
		webrtcPacket := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				Marker:         i == numFragments-1, // 最后一个分片设置marker位
				PayloadType:    96,
				SequenceNumber: rtspPacket.SequenceNumber + uint16(i),
				Timestamp:      webrtcTimestamp,
				SSRC:           rtspPacket.SSRC,
			},
			Payload: make([]byte, 2+end-start),
		}

		// 复制FU-A头和分片数据
		copy(webrtcPacket.Payload[0:2], fuHeader)
		copy(webrtcPacket.Payload[2:], nalu[start:end])

		*packets = append(*packets, webrtcPacket)
	}

	return nil
}

// convertTimestamp 转换时间戳
func (c *RTSPWebRTCConverter) convertTimestamp(rtspTimestamp uint32) uint32 {
	if c.rtspBaseTimestamp == 0 || c.webrtcBaseTimestamp == 0 {
		return rtspTimestamp
	}

	// 计算时间戳增量
	rtspDelta := c.calculateTimestampDelta(rtspTimestamp, c.rtspBaseTimestamp)

	// 转换为WebRTC时间戳
	webrtcTimestamp := c.webrtcBaseTimestamp + rtspDelta

	return webrtcTimestamp
}

// calculateTimestampDelta 计算时间戳差异（处理回绕）
func (c *RTSPWebRTCConverter) calculateTimestampDelta(current, base uint32) uint32 {
	diff := int64(current) - int64(base)

	// 处理32位时间戳回绕
	if diff < -0x80000000 {
		return uint32(diff + 0x100000000)
	} else if diff > 0x80000000 {
		return uint32(diff - 0x100000000)
	}

	return uint32(diff)
}

// sendToWebRTCClients 发送到所有WebRTC客户端
func (c *RTSPWebRTCConverter) sendToWebRTCClients(packets []*rtp.Packet) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	for id, client := range clients {
		for _, packet := range packets {
			if err := client.videoTrack.WriteRTP(packet); err != nil {
				log.Printf("Failed to send RTP packet to client %s: %v", id, err)
				// 客户端可能已断开，稍后会被清理
			}
		}

		c.stats.mutex.Lock()
		c.stats.WebRTCPacketsSent += len(packets)
		c.stats.LastWebRTCTimestamp = packets[len(packets)-1].Timestamp
		c.stats.mutex.Unlock()
	}
}

// createWebRTCClient 创建WebRTC客户端
func createWebRTCClient(clientID string) (*WebRTCClient, error) {
	// 配置WebRTC
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// 创建PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// 创建视频轨道
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType:  "video/H264",
		ClockRate: 90000,
	}, "video", "stream")
	if err != nil {
		return nil, fmt.Errorf("failed to create video track: %w", err)
	}

	// 添加轨道到PeerConnection
	if _, err := peerConnection.AddTrack(videoTrack); err != nil {
		return nil, fmt.Errorf("failed to add video track: %w", err)
	}

	// 处理ICE连接状态变化
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("Client %s ICE state: %s", clientID, state)

		if state == webrtc.ICEConnectionStateDisconnected ||
			state == webrtc.ICEConnectionStateFailed ||
			state == webrtc.ICEConnectionStateClosed {
			removeWebRTCClient(clientID)
		}
	})

	// 处理RTCP接收
	//peerConnection.OnRTCP(func(packets []rtcp.Packet, receiver *webrtc.RTPReceiver) {
	//	handleRTCP(packets, clientID)
	//})

	client := &WebRTCClient{
		ID:             clientID,
		peerConnection: peerConnection,
		videoTrack:     videoTrack,
		sendPLITimer:   time.NewTicker(5 * time.Second),
	}

	// 定期发送PLI请求获取关键帧
	go func() {
		defer client.sendPLITimer.Stop()
		for range client.sendPLITimer.C {
			if client.peerConnection.ConnectionState() == webrtc.PeerConnectionStateConnected {
				//pli := rtcp.PictureLossIndication{MediaSSRC: videoTrack.SSRC()}
				//client.peerConnection.WriteRTCP([]rtcp.Packet{&pli})
			}
		}
	}()

	return client, nil
}

// removeWebRTCClient 移除WebRTC客户端
func removeWebRTCClient(clientID string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	if client, exists := clients[clientID]; exists {
		client.peerConnection.Close()
		client.sendPLITimer.Stop()
		delete(clients, clientID)
		log.Printf("WebRTC client %s removed", clientID)
	}
}

// handleRTCP 处理RTCP包
func handleRTCP(packets []rtcp.Packet, clientID string) {
	for _, packet := range packets {
		switch packet.(type) {
		case *rtcp.PictureLossIndication:
			log.Printf("Received PLI from client %s, requesting key frame", clientID)
			// 这里可以触发关键帧请求
		case *rtcp.ReceiverReport:
			// 处理接收者报告
		}
	}
}

// HTTP处理函数
func handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		RTSPURL string `json:"rtspUrl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	converterMutex.Lock()
	defer converterMutex.Unlock()

	if converter != nil && converter.isRunning {
		http.Error(w, "Converter already running", http.StatusConflict)
		return
	}

	converter = NewRTSPWebRTCConverter(request.RTSPURL)

	err := converter.Start()
	if err != nil {
		http.Error(w, "Failed to start converter: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "started",
		"message": "RTSP to WebRTC conversion started",
	})
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	converterMutex.Lock()
	defer converterMutex.Unlock()

	if converter != nil {
		converter.Stop()
		converter = nil
	}

	// 清理所有WebRTC客户端
	clientsMutex.Lock()
	for id, client := range clients {
		client.peerConnection.Close()
		client.sendPLITimer.Stop()
		delete(clients, id)
	}
	clientsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "stopped",
		"message": "RTSP to WebRTC conversion stopped",
	})
}

func handleOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		SDP string `json:"sdp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 检查转换器是否运行
	converterMutex.Lock()
	if converter == nil || !converter.isRunning {
		converterMutex.Unlock()
		http.Error(w, "Converter not running", http.StatusServiceUnavailable)
		return
	}
	converterMutex.Unlock()

	// 创建客户端ID
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())

	// 创建WebRTC客户端
	client, err := createWebRTCClient(clientID)
	if err != nil {
		http.Error(w, "Failed to create WebRTC client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置远程描述
	remoteOffer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  request.SDP,
	}

	if err := client.peerConnection.SetRemoteDescription(remoteOffer); err != nil {
		http.Error(w, "Failed to set remote description: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 创建answer
	answer, err := client.peerConnection.CreateAnswer(nil)
	if err != nil {
		http.Error(w, "Failed to create answer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := client.peerConnection.SetLocalDescription(answer); err != nil {
		http.Error(w, "Failed to set local description: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 存储客户端
	clientsMutex.Lock()
	clients[clientID] = client
	clientsMutex.Unlock()

	// 发送answer
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sdp":      answer.SDP,
		"clientId": clientID,
	})

	log.Printf("New WebRTC client connected: %s", clientID)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	converterMutex.Lock()
	defer converterMutex.Unlock()

	stats := map[string]interface{}{
		"converterRunning": converter != nil && converter.isRunning,
		"clientsCount":     0,
	}

	if converter != nil {
		converter.stats.mutex.RLock()
		stats["conversionStats"] = converter.stats
		converter.stats.mutex.RUnlock()
	}

	clientsMutex.RLock()
	stats["clientsCount"] = len(clients)
	clientsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func main() {
	// 设置HTTP路由
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/stop", handleStop)
	http.HandleFunc("/offer", handleOffer)
	http.HandleFunc("/stats", handleStats)

	// 启动HTTP服务器
	log.Println("Starting HTTP server on :8080")
	log.Println("Visit http://localhost:8080 to view the converter")

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal("HTTP server error:", err)
	}
}
