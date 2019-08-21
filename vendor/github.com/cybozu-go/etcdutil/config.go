package etcdutil

const (
	// DefaultTimeout is default etcd connection timeout.
	DefaultTimeout = "2s"
)

var (
	// DefaultEndpoints is default etcd servers.
	DefaultEndpoints = []string{"http://127.0.0.1:2379"}
)

// Config represents configuration parameters to access etcd.
type Config struct {
	// Endpoints are etcd servers.
	Endpoints []string `json:"endpoints" toml:"endpoints"`
	// Prefix is etcd prefix key.
	Prefix string `json:"prefix" toml:"prefix"`
	// Timeout is dial timeout of the etcd client connection.
	Timeout string `json:"timeout" toml:"timeout"`
	// Username is username for loging in to the etcd.
	Username string `json:"username" toml:"username"`
	// Password is password for loging in to the etcd.
	Password string `json:"password" toml:"password"`
	// TLSCAFile is root CA path.
	TLSCAFile string `json:"tls-ca-file" toml:"tls-ca-file"`
	// TLSCA is root CA raw string.
	TLSCA string `json:"tls-ca" toml:"tls-ca"`
	// TLSCertFile is TLS client certificate path.
	TLSCertFile string `json:"tls-cert-file" toml:"tls-cert-file"`
	// TLSCert is TLS client certificate raw string.
	TLSCert string `json:"tls-cert" toml:"tls-cert"`
	// TLSKeyFile is TLS client private key.
	TLSKeyFile string `json:"tls-key-file" toml:"tls-key-file"`
	// TLSKey is TLS client private key raw string.
	TLSKey string `json:"tls-key" toml:"tls-key"`
}

// NewConfig creates Config with default values.
func NewConfig(prefix string) *Config {
	return &Config{
		Endpoints: DefaultEndpoints,
		Prefix:    prefix,
		Timeout:   DefaultTimeout,
	}
}
