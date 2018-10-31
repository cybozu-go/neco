package neco

type UpdateRequest struct {
	Version string `json:"version"`
	Servers []int  `json:"servers"`
	Stop    bool   `json:"stop"`
}

type UpdateStatus struct {
	Version  string `json:"version"`
	Finished bool   `json:"finished"`
	Error    bool   `json:"error"`
	Message  string `json:"message"`
}
