package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"sorcia/setting"
	"strings"

	"golang.org/x/crypto/ssh"
)

func cleanCommand(cmd string) string {
	i := strings.Index(cmd, "git")
	if i == -1 {
		return cmd
	}
	return cmd[i:]
}

// RunSSH ...
func RunSSH(conf *setting.BaseStruct) {
	// Public key authentication is done by comparing
	// the public key of a received connection
	// with the entries in the authorized_keys file.
	authorizedKeysBytes, err := ioutil.ReadFile("C:\\Users\\mysticmode\\.ssh\\authorized_keys")
	if err != nil {
		log.Fatalf("Failed to load authorized_keys, err: %v", err)
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			log.Fatal(err)
		}

		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}

	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	privateBytes, err := ioutil.ReadFile("C:\\Users\\mysticmode\\.ssh\\id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:1938")
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	nConn, err := listener.Accept()
	if err != nil {
		log.Fatal("failed to accept incoming connection: ", err)
	}

	// Before use, a handshake must be performed on the incoming
	// net.Conn.
	conn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Fatal("failed to handshake: ", err)
	}
	log.Printf("logged in with key %s", conn.Permissions.Extensions["pubkey-fp"])

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple
		// terminal interface.
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Fatalf("Could not accept channel: %v", err)
		}

		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env". Here we handle only the
		// "shell" request.
		go func(in <-chan *ssh.Request) {
			defer channel.Close()

			for req := range in {
				payload := cleanCommand(string(req.Payload))
				fmt.Println(req.Type)

				cmdName := strings.TrimLeft(payload, "'()")
				fmt.Println("cmdName")
				fmt.Println(cmdName)

				cmd := exec.Command(strings.Split(cmdName, " ")[0], "joyread.git")
				cmd.Dir = "D:\\Work\\sorcia\\repositories\\+mysticmode"

				stdout, err := cmd.StdoutPipe()
				if err != nil {
					log.Printf("ssh: cant open stdout pipe: %v", err)
					return
				}

				stderr, err := cmd.StderrPipe()
				if err != nil {
					log.Printf("ssh: cant open stderr pipe: %v", err)
					return
				}

				input, err := cmd.StdinPipe()
				if err != nil {
					log.Printf("ssh: cant open stdin pipe: %v", err)
					return
				}

				if err = cmd.Start(); err != nil {
					log.Printf("ssh: start error: %v", err)
					return
				}

				req.Reply(true, nil)
				go io.Copy(input, channel)
				io.Copy(channel, stdout)
				io.Copy(channel.Stderr(), stderr)

				if err = cmd.Wait(); err != nil {
					log.Printf("ssh: command failed: %v", err)
					return
				}

				channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})

				return
			}
		}(requests)
	}
}
