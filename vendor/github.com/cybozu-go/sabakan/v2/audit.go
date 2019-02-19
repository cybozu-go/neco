package sabakan

import (
	"context"
	"time"
)

// AuditCategory is the type of audit categories.
type AuditCategory string

// Audit categories.
const (
	AuditAssets   = AuditCategory("assets")
	AuditCrypts   = AuditCategory("crypts")
	AuditDHCP     = AuditCategory("dhcp")
	AuditIgnition = AuditCategory("ignition")
	AuditImage    = AuditCategory("image")
	AuditIPAM     = AuditCategory("ipam")
	AuditIPXE     = AuditCategory("ipxe")
	AuditMachines = AuditCategory("machines")
)

// AuditLog represents an audit log entry.
type AuditLog struct {
	Timestamp time.Time     `json:"ts"`
	Revision  int64         `json:"rev,string"`
	User      string        `json:"user"`
	IP        string        `json:"ip"`
	Host      string        `json:"host"`
	Category  AuditCategory `json:"category"`
	Instance  string        `json:"instance"`
	Action    string        `json:"action"`
	Detail    string        `json:"detail"`
}

// AuditContextKey is the type of context keys for audit.
type AuditContextKey string

// Audit context keys.  Values must be string.
const (
	AuditKeyUser = AuditContextKey("user")
	AuditKeyIP   = AuditContextKey("ip")
	AuditKeyHost = AuditContextKey("host")
)

// NewAuditLog creates an audit log entry and initializes it.
func NewAuditLog(ctx context.Context, ts time.Time, rev int64, cat AuditCategory,
	instance, action, detail string) *AuditLog {

	a := new(AuditLog)
	a.Timestamp = ts.UTC()
	a.Revision = rev
	if v := ctx.Value(AuditKeyUser); v != nil {
		a.User = v.(string)
	}
	if v := ctx.Value(AuditKeyIP); v != nil {
		a.IP = v.(string)
	}
	if v := ctx.Value(AuditKeyHost); v != nil {
		a.Host = v.(string)
	}
	a.Category = cat
	a.Instance = instance
	a.Action = action
	a.Detail = detail

	return a
}
