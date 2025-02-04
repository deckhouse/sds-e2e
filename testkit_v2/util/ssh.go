package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func NewSSHClient(user string, addr string, port uint, key string) (client *goph.Client) {

	auth, err := goph.Key(key, "")
	if err != nil {
		log.Fatal(err.Error())
	}

	//callback, err := goph.DefaultKnownHosts()
	//if err != nil {
	//	log.Fatal(err.Error())
	//	return
	//}

	cfg := goph.Config{
		User:     user,
		Addr:     addr,
		Port:     port,
		Auth:     auth,
		Timeout:  0, //20 * time.Second,
		Callback: ssh.InsecureIgnoreHostKey(), //callback,
	}

	for count := 0; count < retries; count++ {
		if client, err = goph.NewConn(&cfg); err == nil {
			break
		}

		time.Sleep(10 * time.Second)

		if count == retries-1 {
			log.Fatal("Timeout waiting for installer VM to be ready")
		}
	}

	return
}

func GetSSHClient(ip string, username string, auth goph.Auth) *goph.Client {
	var client *goph.Client
	var err error

	for count := 0; count < retries; count++ {
		client, err = goph.NewUnknown(username, ip, auth)
		if err == nil {
			break
		}

		time.Sleep(10 * time.Second)

		if count == retries-1 {
			log.Fatal("Timeout waiting for installer VM to be ready")
		}
	}

	return client
}

func ExecSshFatal(client *goph.Client, command string) string {
	out, err := client.Run(command)
	if err != nil {
		Infof(command)
		Fatalf(err.Error())
	}
	return string(out)
}

func ExecSshCheck(client *goph.Client, command string, outContains []string) string {
	out := ExecSshFatal(client, command)
	for _, contains := range outContains {
		if !strings.Contains(out, contains) {
			Infof(command)
			Infof(out)
			Fatalf("No '%s' in output", contains)
		}
	}
	return out
}
