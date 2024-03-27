package funcs

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"net"
)

func RemoteRun(user string, addr string, password string, cmd string) (string, error) {
	// privateKey could be read from a file, or retrieved from another storage
	// source, such as the Secret Service / GNOME Keyring
	//key, err := ssh.ParsePrivateKey([]byte(privateKey))
	//if err != nil {
	//	return "", err
	//}
	// Authentication
	config := &ssh.ClientConfig{
		User: user,
		// https://github.com/golang/go/issues/19767
		// as clientConfig is non-permissive by default
		// you can set ssh.InsercureIgnoreHostKey to allow any host
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		//Auth: []ssh.AuthMethod{
		//	ssh.PublicKeys(key),
		//},
		//alternatively, you could use a password
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		Timeout: 0,
	}
	// Connect
	client, err := ssh.Dial("tcp", net.JoinHostPort(addr, "22"), config)
	if err != nil {
		return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var b bytes.Buffer  // import "bytes"
	session.Stdout = &b // get output
	// you can also pass what gets input to the stdin, allowing you to pipe
	// content from client to server
	//      session.Stdin = bytes.NewBufferString("My input")

	// Finally, run the command
	err = session.Run(cmd)
	return b.String(), err
}
