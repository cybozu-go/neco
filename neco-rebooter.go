package neco

type RebootListEntry struct {
	Index      int64  `json:"index"`
	Node       string `json:"node"`
	Group      string `json:"group"`
	RebootTime string `json:"reboot_time"`
	Status     string `json:"status"`
}

var (
	RebootListEntryStatusPending   = "Pending"
	RebootListEntryStatusQueued    = "Queued"
	RebootListEntryStatusCancelled = "Cancelled"
)
