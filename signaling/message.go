package signaling

type Message struct {
	Id   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Data string `json:"data,omitempty"`
}

type Connect struct {
	Url     string         `json:"url"`
	Options map[string]any `json:"options,omitempty"`
}
