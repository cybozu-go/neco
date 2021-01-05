package etcdpasswd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/neco"
)

const (
	sshdConfFile = "/etc/ssh/sshd_config.d/neco.conf"
	sshdConf     = `# SSHD configurations for Neco

AuthorizedKeysFile	.ssh/authorized_keys
PasswordAuthentication no
`
)

// TODO: remove this when Ubuntu 18.04 support is dropped.
const sshdConfBionic = `#	$OpenBSD: sshd_config,v 1.101 2017/03/14 07:19:07 djm Exp $

# This is the sshd server system-wide configuration file.  See
# sshd_config(5) for more information.

# This sshd was compiled with PATH=/usr/bin:/bin:/usr/sbin:/sbin

# The strategy used for options in the default sshd_config shipped with
# OpenSSH is to specify options with their default value where
# possible, but leave them commented.  Uncommented options override the
# default value.

#Port 22
#AddressFamily any
#ListenAddress 0.0.0.0
#ListenAddress ::

#HostKey /etc/ssh/ssh_host_rsa_key
#HostKey /etc/ssh/ssh_host_ecdsa_key
#HostKey /etc/ssh/ssh_host_ed25519_key

# Ciphers and keying
#RekeyLimit default none

# Logging
#SyslogFacility AUTH
#LogLevel INFO

# Authentication:

#LoginGraceTime 2m
#PermitRootLogin prohibit-password
#StrictModes yes
#MaxAuthTries 6
#MaxSessions 10

# Cybozu: NecoTask-220
PubkeyAuthentication yes

# Expect .ssh/authorized_keys2 to be disregarded by default in future.
# Cybozu: NecoTask-220
AuthorizedKeysFile	.ssh/authorized_keys

#AuthorizedPrincipalsFile none

#AuthorizedKeysCommand none
#AuthorizedKeysCommandUser nobody

# For this to work you will also need host keys in /etc/ssh/ssh_known_hosts
#HostbasedAuthentication no
# Change to yes if you don't trust ~/.ssh/known_hosts for
# HostbasedAuthentication
#IgnoreUserKnownHosts no
# Don't read the user's ~/.rhosts and ~/.shosts files
#IgnoreRhosts yes

# To disable tunneled clear text passwords, change to no here!
# Cybozu: NecoTask-220
PasswordAuthentication no
#PermitEmptyPasswords no

# Change to yes to enable challenge-response passwords (beware issues with
# some PAM modules and threads)
ChallengeResponseAuthentication no

# Kerberos options
#KerberosAuthentication no
#KerberosOrLocalPasswd yes
#KerberosTicketCleanup yes
#KerberosGetAFSToken no

# GSSAPI options
#GSSAPIAuthentication no
#GSSAPICleanupCredentials yes
#GSSAPIStrictAcceptorCheck yes
#GSSAPIKeyExchange no

# Set this to 'yes' to enable PAM authentication, account processing,
# and session processing. If this is enabled, PAM authentication will
# be allowed through the ChallengeResponseAuthentication and
# PasswordAuthentication.  Depending on your PAM configuration,
# PAM authentication via ChallengeResponseAuthentication may bypass
# the setting of "PermitRootLogin without-password".
# If you just want the PAM account and session checks to run without
# PAM authentication, then enable this but set PasswordAuthentication
# and ChallengeResponseAuthentication to 'no'.
UsePAM yes

#AllowAgentForwarding yes
#AllowTcpForwarding yes
#GatewayPorts no
X11Forwarding yes
#X11DisplayOffset 10
#X11UseLocalhost yes
#PermitTTY yes
PrintMotd no
#PrintLastLog yes
#TCPKeepAlive yes
#UseLogin no
#PermitUserEnvironment no
#Compression delayed
#ClientAliveInterval 0
#ClientAliveCountMax 3
#UseDNS no
#PidFile /var/run/sshd.pid
#MaxStartups 10:30:100
#PermitTunnel no
#ChrootDirectory none
#VersionAddendum none

# no default banner path
#Banner none

# Allow client to pass locale environment variables
AcceptEnv LANG LC_*

# override default of no subsystems
Subsystem	sftp	/usr/lib/openssh/sftp-server

# Example of overriding settings on a per-user basis
#Match User anoncvs
#	X11Forwarding no
#	AllowTcpForwarding no
#	PermitTTY no
#	ForceCommand cvs server
`

// InstallSshdConf installs sshd_config file for Neco
func InstallSshdConf() error {
	codename, err := neco.OSCodename()
	if err != nil {
		return err
	}

	if codename == "bionic" {
		err := ioutil.WriteFile("/etc/ssh/sshd_config", []byte(sshdConfBionic), 0644)
		if err != nil {
			return fmt.Errorf("failed to write to /etc/ssh/sshd_config: %w", err)
		}
		return nil
	}

	tmpFile := sshdConfFile + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", tmpFile, err)
	}
	defer f.Close()

	if _, err := f.Write([]byte(sshdConf)); err != nil {
		return fmt.Errorf("failed to write to %s: %w", tmpFile, err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to fsync %s: %w", tmpFile, err)
	}
	if err := os.Rename(tmpFile, sshdConfFile); err != nil {
		return fmt.Errorf("failed to rename %s: %w", tmpFile, err)
	}
	return nil
}
