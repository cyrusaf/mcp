package rpc

// InitializeResult describes the response payload for the JSON-RPC "initialize" call.
type InitializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Capabilities struct {
		Tools struct {
			ListChanged bool `json:"listChanged"`
		} `json:"tools"`
		Resources struct {
			ListChanged bool `json:"listChanged"`
			Subscribe   bool `json:"subscribe"`
		} `json:"resources"`
		Prompts struct {
			Offered bool `json:"offered"`
		} `json:"prompts"`
	} `json:"capabilities"`
}
