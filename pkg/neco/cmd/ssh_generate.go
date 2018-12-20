package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

const rsaBits = 2048

var sshGenerateDumpPrivateKey bool

func makeSSHKeyPair() (pubkey, privkey []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privkey = pem.EncodeToMemory(privateKeyPEM)

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	pubkey = ssh.MarshalAuthorizedKey(pub)
	return
}

// sshGenerateCmd implements "ssh generate".
var sshGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate a new SSH key pair for sabakan machines",
	Long: `Generates a new SSH key pair for sabakan controlled machines.

The generated public key is stored in etcd.
The generated private key is stored in Vault by using
"ckecli vault ssh-privkey".
`,

	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()

		pubkey, privkey, err := makeSSHKeyPair()
		if err != nil {
			log.ErrorExit(err)
		}

		well.Go(func(ctx context.Context) error {
			st := storage.NewStorage(etcd)
			if err := st.PutSSHPubkey(ctx, string(pubkey)); err != nil {
				return err
			}

			cmd := well.CommandContext(ctx, neco.CKECLIBin, "vault", "ssh-privkey", "-")
			cmd.Stdin = bytes.NewReader(privkey)
			if err := cmd.Run(); err != nil {
				return err
			}

			if sshGenerateDumpPrivateKey {
				fmt.Println(string(privkey))
			}
			return nil
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	sshGenerateCmd.Flags().BoolVar(&sshGenerateDumpPrivateKey, "dump", false, "dump generated private key to stdout")
	sshCmd.AddCommand(sshGenerateCmd)
}
