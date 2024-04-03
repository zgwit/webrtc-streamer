package signaling

type Message struct {
	Id   string `json:"client,omitempty"`
	Type string `json:"type,omitempty"`
	Data string `json:"data,omitempty"`
}
