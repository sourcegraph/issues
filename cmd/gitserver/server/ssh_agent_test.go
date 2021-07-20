package server

import (
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/encryption"
)

func TestSSHAgent(t *testing.T) {
	// Generate a keypair to use for the client-server connection.
	keypair, err := encryption.GenerateRSAKey()
	if err != nil {
		t.Fatal(err)
	}

	// Spawn the agent using the keypair from above.
	a, err := newSSHAgent([]byte(keypair.PrivateKey), []byte(keypair.Passphrase))
	if err != nil {
		t.Fatal(err)
	}
	go a.Listen()
	defer a.Close()

	// Spawn an ssh server which will accept the public key from the keypair.
	authorizedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keypair.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			// If the public keys match, grant access.
			if string(pubKey.Marshal()) == string(authorizedKey.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, errors.Errorf("unknown public key for %q", c.User())
		},
	}
	serverKey, err := encryption.GenerateRSAKey()
	if err != nil {
		t.Fatal(err)
	}
	decryptedServerKey, err := ssh.ParsePrivateKeyWithPassphrase([]byte(serverKey.PrivateKey), []byte(serverKey.Passphrase))
	if err != nil {
		t.Fatal(err)
	}
	serverConfig.AddHostKey(decryptedServerKey)
	// Listen on a random available port.
	serverListener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer serverListener.Close()
	errs := make(chan error)
	defer close(errs)
	done := make(chan struct{})
	defer close(done)
	go func() {
		netConn, err := serverListener.Accept()
		if err != nil {
			select {
			case <-done:
			default:
				errs <- err
			}
			return
		}
		defer netConn.Close()
		conn, chans, reqs, err := ssh.NewServerConn(netConn, serverConfig)
		if err != nil {
			errs <- err
			return
		}
		defer conn.Close()
		go ssh.DiscardRequests(reqs)
		for newChannel := range chans {
			// Accept and goodbye.
			channel, _, err := newChannel.Accept()
			if err != nil {
				errs <- err
				return
			}
			channel.Close()
		}
	}()

	// Now try to connect to that server using the private key from the keypair.
	agentConn, err := net.Dial("unix", a.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer agentConn.Close()
	agentClient := agent.NewClient(agentConn)
	clientConfig := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {
			// Talk to the ssh-agent to get the signers.
			return agentClient.Signers()
		})},
	}
	client, err := ssh.Dial("tcp", serverListener.Addr().String(), clientConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	select {
	// Check if the server errored.
	case err := <-errs:
		if err != nil {
			t.Fatal(err)
		}
	default:
		return
	}
}
