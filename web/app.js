let stream = new MediaStream()
let pc = new RTCPeerConnection()

let ws = new WebSocket("ws://localhost:8080/connect/test")
let id

function send(type, data) {
    if (typeof data === "object") {
        data = JSON.stringify(data)
    }
    ws.send(JSON.stringify({type, data}))
}

ws.onerror = console.error

ws.onopen = function (event) {
    console.log("websocket onopen")
    send("connect", {
        url: "rtsp://localhost:8554/mystream"
    })
}

ws.onmessage = function (event) {
    console.log("onmessage", event)
    let data = event.data
    let msg = JSON.parse(data)
    id = msg.id
    switch (msg.type) {
        case "answer":
            pc.setRemoteDescription(new RTCSessionDescription({type: 'answer', sdp: msg.data})).then()
            break
        case "candidate":
            pc.addIceCandidate(new RTCIceCandidate(JSON.parse(msg.data))).then()
            break
    }
}

pc.onnegotiationneeded = async function () {
    console.log("onnegotiationneeded")

    let offer = await pc.createOffer()
    await pc.setLocalDescription(offer)
    send("offer", offer.sdp)
};


pc.ontrack = function (event) {
    console.log("ontrack", event.streams.length + ' track is delivered')

    stream.addTrack(event.track);
    let videoElem = document.getElementById("video")
    videoElem.srcObject = stream;
}

pc.onicecandidate = function (event) {
    send("candidate", event.candidate.toJSON())
}

pc.oniceconnectionstatechange = function (event) {
    console.log("oniceconnectionstatechange", pc.iceConnectionState)
}


// pc.addTransceiver(value.Type, {
//     'direction': 'sendrecv'
// })
