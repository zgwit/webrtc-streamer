# webrtc-streamer

Golang实现的WebRTC推流工具，基于大名鼎鼎的pion/webrtc库，支持监控摄像头 和 usb摄像头。

```
go get -u github.com/zgwit/webrtc-streamer
```

项目灵感来源[mpromonet/webrtc-streamer](https://github.com/mpromonet/webrtc-streamer)，
此前使用它做了视频直播的引擎，但是由于没有分发功能，多路观看时会重复调用编码器，导致CPU占用过高。
另外，用C++实现虽然很高效，但是乱七八糟、随心所欲的代码，根本没办法借鉴，十多个第三方依赖，编译一次十分痛苦。
所以，我使用了更简洁的golang进行重新实现，名字也叫webrtc-streamer，蹭蹭流量。

项目实现了独立的信令服务器，推流端可以放在内网，实现p2p直播更方便！

推流器支持正向握手，反向握手

ICE交换使用trickle ice模式，速度更快（平均1.5秒）

ps. 项目主要是为实现物联大师（物联网云平台）的视频监控远程接入功能，有兴趣的小伙伴可以去看看，顺便加个星。

[github.com/zgwit/iot-master](https://github.com/zgwit/iot-master)

## 开发进度

- [x] 信令服务器
- [x] WebRTC推流
- [x] 视频分发
- [x] rtsp视频 h264
- [ ] rtsp视频 h265（已实现，待验证）
- [ ] rtsp转码 （格式，压缩，抽帧等）
- [ ] rtsp音频
- [ ] 摄像头
- [ ] USB摄像头和采集卡（同上）
- [ ] 视频文件循环播放
- [ ] 云台控制

## 开源协议

MIT
