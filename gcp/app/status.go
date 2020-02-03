package app

// ShutdownStatus represents status of /shutdown
type ShutdownStatus struct {
	Stopped []string `json:"stopped"`
	Deleted []string `json:"deleted"`
}

type ExtendStatus struct {
	Extended string `json:"extended"`
}
