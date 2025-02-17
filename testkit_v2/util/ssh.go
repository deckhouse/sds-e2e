package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// generatePrivateKey creates an RSA Private Key of specified byte size
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	log.Println("Private Key generated")
	return privateKey, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privateDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privateBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privateBlock)

	return privatePEM
}

// generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// returns in the format "ssh-rsa ..."
func generatePublicKey(privateKey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privateKey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	log.Println("Public key generated")
	return pubKeyBytes, nil
}

// writePemToFile writes keys to a file
func writeKeyToFile(keyBytes []byte, saveFileTo string) error {
	err := os.WriteFile(saveFileTo, keyBytes, 0600)
	if err != nil {
		return err
	}

	log.Printf("Key saved to: %s", saveFileTo)
	return nil
}

func GenerateRSAKeys(privateFilename string, publicFilename string) {
	if _, err := os.Stat(privateFilename); err == nil {
		return
	}

	bitSize := 4096
	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		log.Fatal(err.Error())
	}

	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatal(err.Error())
	}

	privateKeyBytes := encodePrivateKeyToPEM(privateKey)

	err = writeKeyToFile(privateKeyBytes, privateFilename)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = writeKeyToFile(publicKeyBytes, publicFilename)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func CheckAndGetSSHKeys(dir string, privateKeyName string, pubKeyName string) (sshPubKeyString string) {
	GenerateRSAKeys(filepath.Join(dir, privateKeyName), filepath.Join(dir, pubKeyName))

	sshPubKey, err := os.ReadFile(filepath.Join(dir, pubKeyName))
	if err != nil {
		log.Fatal(err.Error())
	}

	return string(sshPubKey)
}

func readPassword(prompt string) ([]byte, error) {
	if !testing.Verbose() {
		panic("Can't read password in not verbose mode")
	}

	fmt.Fprint(os.Stderr, prompt)
	var fd int
	if terminal.IsTerminal(syscall.Stdin) {
		fd = syscall.Stdin
	} else {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return nil, fmt.Errorf("error allocating terminal")
		}
		defer tty.Close()
		fd = int(tty.Fd())
	}

	pass, err := terminal.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	return pass, err
}

func newSshPassConfig(user string) *ssh.ClientConfig {
	pass, err := readPassword("    Enter ssh password: ")
	if err != nil {
		Fatalf("unable to get ssh password: %s", err.Error())
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pass)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}
}

func newSshConfig(user, keyPath string) *ssh.ClientConfig {
	if keyPath == "" {
		return newSshPassConfig(user)
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		Fatalf("unable to read private key: %s", err.Error())
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if err.Error() != "ssh: this private key is passphrase protected" {
			Fatalf("unable to parse private key: %s", err.Error())
		}

		pass, err := readPassword("    Enter passphrase for '" + keyPath + "': ")
		if err != nil {
			Fatalf("unable to get ssh password: %s", err.Error())
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, pass)
		if err != nil {
			Fatalf("unable to parse private key: %s", err.Error())
		}
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}
}

type sshClient struct {
	client *ssh.Client
}

func GetSshClient(user, addr, keyPath string) sshClient {
	config := newSshConfig(user, keyPath)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		Fatalf("Ssh Dial error: %s", err.Error())
	}

	return sshClient{client: client}
}

func (c sshClient) Close() error {
	return c.client.Close()
}

func (c sshClient) Upload(localPath string, remotePath string) error {
	local, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer local.Close()

	ftp, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer ftp.Close()

	remote, err := ftp.Create(remotePath)
	if err != nil {
		return err
	}
	defer remote.Close()

	_, err = io.Copy(remote, local)
	return err
}

func (c sshClient) Download(remotePath string, localPath string) error {
	local, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer local.Close()

	ftp, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer ftp.Close()

	remote, err := ftp.Open(remotePath)
	if err != nil {
		return err
	}
	defer remote.Close()

	if _, err = io.Copy(local, remote); err != nil {
		return err
	}

	return local.Sync()
}

func (c sshClient) Dial(n, addr string) (net.Conn, error) {
	for i := 0; ; i++ {
		conn, err := c.client.Dial(n, addr)
		if err == nil {
			return conn, nil
		}

		if i >= retries {
			Fatalf("Fwd Dial '%s' error: %s", addr, err.Error())
		}
		time.Sleep(10 * time.Second)
	}

	return nil, nil
}

func (c sshClient) GetFwdClient(user, addr, keyPath string) sshClient {
	conn, _ := c.Dial("tcp", addr)

	config := newSshConfig(user, keyPath)
	ncc, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		Fatalf("NewClientConn '%s@%s' error: %s", user, addr, err.Error())
	}

	return sshClient{client: ssh.NewClient(ncc, chans, reqs)}
}

func (c sshClient) NewTunnel(lAddr, rAddr string) {
	listener, err := net.Listen("tcp", lAddr)
	if err != nil {
		Fatalf("New tunnel Listen error: %s", err.Error())
	}
	defer listener.Close()

	for {
		// one local connections at a time
		local, err := listener.Accept()
		if err != nil {
			Fatalf("Accept listener error: %s", err.Error())
		}
		defer local.Close()

		remote, _ := c.Dial("tcp", rAddr)
		defer remote.Close()

		done := make(chan struct{})

		go func() {
			if _, err = io.Copy(local, remote); err != nil {
				Errf("Copy local->remote error: %s", err.Error())
			}
			done <- struct{}{}
		}()

		go func() {
			if _, err = io.Copy(remote, local); err != nil {
				Errf("Copy remote->local error: %s", err.Error())
			}
			done <- struct{}{}
		}()

		<-done
	}
}

func (c sshClient) Exec(cmd string) (string, error) {
	sess, err := c.client.NewSession()
	if err != nil {
		return "", err
	}

	defer sess.Close()

	out, err := sess.CombinedOutput(cmd)
	return string(out), err
}

func (c sshClient) ExecFatal(cmd string) string {
	out, err := c.Exec(cmd)
	if err != nil {
		Fatalf("Exec ssh error: %s", err.Error())
	}
	return out
}
