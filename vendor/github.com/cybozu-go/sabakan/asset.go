package sabakan

import "time"

// Asset represents an asset.
type Asset struct {
	Name        string            `json:"name"`
	ID          int               `json:"id,string"`
	ContentType string            `json:"content-type"`
	Date        time.Time         `json:"date"`
	Sha256      string            `json:"sha256"`
	URLs        []string          `json:"urls"`
	Exists      bool              `json:"exists"`
	Options     map[string]string `json:"options"`
}

// AssetStatus is the status of an asset.
type AssetStatus struct {
	Status int `json:"status"`
	ID     int `json:"id,string"`
}
