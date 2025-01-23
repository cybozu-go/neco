package vault

// AdminPolicy returns policy for admin
func AdminPolicy() string {
	return `# Manage auth methods broadly across Vault
path "auth/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Test
path "sys/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# List, create, update, and delete key/value secrets
path "secret/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
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
