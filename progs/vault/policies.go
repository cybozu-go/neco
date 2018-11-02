package vault

// AdminPolicy returns policy for admin
func AdminPolicy() string {
	return `# Manage auth methods broadly across Vault
path "auth/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# List, create, update, and delete auth methods
path "sys/auth/*"
{
  capabilities = ["create", "read", "update", "delete", "sudo"]
}

# List existing policies
path "sys/policy"
{
  capabilities = ["read"]
}

# Create and manage ACL policies broadly across Vault
path "sys/policy/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Create and manage secret engines broadly across Vault.
path "sys/mounts/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# List existing secret engines.
path "sys/mounts"
{
  capabilities = ["read"]
}

# Read health checks
path "sys/health"
{
  capabilities = ["read", "sudo"]
}
`
}

// CAAdminPolicy returns policy for ca-admin
func CAAdminPolicy() string {
	return `# Manage CA (pki secret engines)
path "ca/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
`
}
