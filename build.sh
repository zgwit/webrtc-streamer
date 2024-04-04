
#编译前端
#npm run build

# 整体编译
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOPRIVATE=*.gitlab.com,*.gitee.com
go env -w GOSUMDB=off


export GOARCH=amd64
export GOOS=windows

go build -o signaling-server.exe server/main.go

go build -o webrtc-streamer.exe main.go
