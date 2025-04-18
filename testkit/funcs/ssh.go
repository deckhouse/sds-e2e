/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package funcs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
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
	savePrivateFileTo := privateFilename
	savePublicFileTo := publicFilename
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

	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = writeKeyToFile(publicKeyBytes, savePublicFileTo)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func CheckAndGetSSHKeys(appTmpPath string, privateKeyName string, pubKeyName string) (sshPubKeyString string) {
	if _, err := os.Stat(filepath.Join(appTmpPath, privateKeyName)); err == nil {
	} else if errors.Is(err, os.ErrNotExist) {
		GenerateRSAKeys(filepath.Join(appTmpPath, privateKeyName), filepath.Join(appTmpPath, pubKeyName))
	}

	sshPubKey, err := os.ReadFile(filepath.Join(appTmpPath, pubKeyName))
	if err != nil {
		log.Fatal(err.Error())
	}

	return string(sshPubKey)
}

func GetSSHClient(ip string, username string, auth goph.Auth) *goph.Client {
	var client *goph.Client
	var err error
	tries := 600
	for count := 0; count < tries; count++ {
		client, err = goph.NewUnknown(username, ip, auth)
		if err == nil {
			break
		}

		time.Sleep(10 * time.Second)

		if count == tries-1 {
			log.Fatal("Timeout waiting for installer VM to be ready")
		}
	}

	return client
}

func ExecuteSSHCommandWithCheck(client *goph.Client, ip string, command string, checkStrings []string) {
	out, err := client.Run(command)
	for _, checkString := range checkStrings {
		if !strings.Contains(string(out), checkString) || err != nil {
			LogFatalIfError(err, fmt.Sprintf("%s error: %s", command, out))
		}
	}
	log.Printf("%s: %s \n%s", ip, command, out)
}
