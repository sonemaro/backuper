package backuper

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// SCPUtil preforms data transfer between a local host and a remote one
type SCPUtil struct {
	// Remote is remote address and port. Format is host:port
	Remote string

	// Path of private key
	PrivateKey string

	// Username of remote server
	Username string
}

// Copy copies a local file to remote server
// We need a know size since we don't want to
// read all data to memory. To find more see client.Copy
func (s *SCPUtil) Copy(src io.Reader, dst string, size int64) error {
	clientConfig, _ := auth.PrivateKey(s.Username, s.PrivateKey, trustedHostKeyCallback(""))
	client := scp.NewClient(s.Remote, &clientConfig)
	err := client.Connect()
	if err != nil {
		return err
	}
	defer client.Close()

	err = client.Copy(src, dst, "0655", size)

	if err != nil {
		return err
	}
	return nil
}

// create human-readable SSH-key strings
func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal()) // e.g. "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY...."
}

func trustedHostKeyCallback(trustedKey string) ssh.HostKeyCallback {

	if trustedKey == "" {
		return func(_ string, _ net.Addr, k ssh.PublicKey) error {
			log.WithFields(log.Fields{
				"trustedKey": keyString(k),
			}).Warn("SSH-key verification is *NOT* in effect: to fix, add this trustedKey: %q")
			return nil
		}
	}

	return func(_ string, _ net.Addr, k ssh.PublicKey) error {
		ks := keyString(k)
		if trustedKey != ks {
			err := fmt.Errorf("SSH-key verification: expected %q but got %q", trustedKey, ks)
			log.WithFields(log.Fields{
				"expected": trustedKey,
				"got":      ks,
			}).Error("SSH-key verification error")
			return err
		}

		return nil
	}
}
