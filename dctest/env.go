package dctest

import (
	"os"
)

var (
	boot0      = os.Getenv("BOOT0")
	boot1      = os.Getenv("BOOT1")
	boot2      = os.Getenv("BOOT2")
	sshKeyFile = os.Getenv("SSH_PRIVKEY")
)
