package worker

import "github.com/zgwit/iot-master/v4/pkg/config"

const MODULE = "webrtc_worker"

func init() {
	config.Register(MODULE, "url", "localhost:8080/worker/")
	config.Register(MODULE, "id", "embed")
}
