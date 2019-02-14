package dctest

import (
	"os"
)

var (
	boot0            = os.Getenv("BOOT0")
	boot1            = os.Getenv("BOOT1")
	boot2            = os.Getenv("BOOT2")
	boot3            = os.Getenv("BOOT3")
	debVer           = os.Getenv("DEBVER")
	generatedDebFile = os.Getenv("GENERATED_DEB")
	baseDebFile      = os.Getenv("BASE_DEB")
	sshKeyFile       = os.Getenv("SSH_PRIVKEY")
	bobPublicKey     = os.Getenv("BOB_PUBKEY")
	bobPrivateKey    = os.Getenv("BOB_PRIVKEY")
)
