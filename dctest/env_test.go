package dctest

import (
	"os"
)

var (
	debVer           = os.Getenv("DEBVER")
	debFile          = os.Getenv("DEB")
	sshKeyFile       = os.Getenv("SSH_PRIVKEY")
	bobPublicKey     = os.Getenv("BOB_PUBKEY")
	bobPrivateKey    = os.Getenv("BOB_PRIVKEY")
	machinesFile     = os.Getenv("MACHINES_FILE")
	addressPoolsFile = os.Getenv("ADDRESS_POOLS_FILE")
	testSuite        = os.Getenv("SUITE")
)

var (
	bootServers    []string
	allBootServers []string
)
