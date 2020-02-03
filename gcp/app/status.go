package app

// ShutdownStatus represents status of /shutdown
type ShutdownStatus struct {
	Stopped []string `json:"stopped"`
	Deleted []string `json:"deleted"`
}

// ExtendStatus represents status of /extend
type ExtendStatus struct {
	Extended string `json:"extended"`
}
