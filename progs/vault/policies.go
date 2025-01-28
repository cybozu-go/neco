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
  capabilities = ["create", "update", "delete", "sudo"]
}

# List auth methods
path "sys/auth"
{
  capabilities = ["read"]
}

# Create and manage ACL policies via CLI
path "sys/policy/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Create and manage ACL policies via API
path "sys/policies/acl/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# To list policies - Step 3
path "sys/policy"
{
  capabilities = ["read"]
}

# To perform Step 4
path "sys/capabilities"
{
  capabilities = ["create", "update"]
}

# To perform Step 4
path "sys/capabilities-self"
{
  capabilities = ["create", "update"]
}

# List, create, update, and delete key/value secrets
path "secret/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Manage secret engines broadly across Vault
path "sys/mounts/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# List existing secret engines
path "sys/mounts"
{
  capabilities = ["read"]
}

# Read health checks
path "sys/health"
{
  capabilities = ["read", "sudo"]
}

# Forces the node to give up active status
path "sys/step-down"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Read the HA status of a Vault cluster
path "sys/ha-status"
{
  capabilities = ["read"]
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
