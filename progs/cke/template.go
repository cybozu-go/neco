package cke

var ckePolicy = `
path "cke/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
`
