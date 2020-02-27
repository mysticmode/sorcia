package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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

func execCommandBytes(cmdname string, args ...string) ([]byte, []byte, error) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cmd := exec.Command(cmdname, args...)
	cmd.Stdout = bufOut
	cmd.Stderr = bufErr

	err := cmd.Run()
	return bufOut.Bytes(), bufErr.Bytes(), err
}

func handleServer(keyID string, sorciaConf *setting.BaseStruct, chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Fatalf("Could not accept channel: %v", err)
		}

		go func(in <-chan *ssh.Request) {
			defer channel.Close()

			for req := range in {
				payload := cleanCommand(string(req.Payload))

				switch req.Type {

				case "env":
					args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
					if len(args) != 2 {
						log.Printf("env: invalid env arguments: '%#v'", args)
						continue
					}

					args[0] = strings.TrimLeft(args[0], "\x04")

					_, _, err := execCommandBytes("env", args[0]+"="+args[1])
					if err != nil {
						log.Printf("env: %v", err)
						return
					}

				case "exec":
					cmdName := strings.TrimLeft(payload, "'()")
					gitRPC := strings.Split(cmdName, " ")[0]
					repoName := strings.Split(cmdName, " ")[1]
					cmd := exec.Command(gitRPC, repoName)
					cmd.Dir = sorciaConf.Paths.RepoPath

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

				default:
					channel.Write([]byte("Unsupported request type.\r\n"))
					log.Println("ssh: unsupported req type:", req.Type)
					return
				}

			}
		}(requests)
	}
}

func runSSH(config *ssh.ServerConfig, sorciaConf *setting.BaseStruct, host, port string) {
	listener, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection: ", err)
		}

		// Before use, a handshake must be performed on the incoming
		// net.Conn.
		go func() {
			conn, chans, reqs, err := ssh.NewServerConn(nConn, config)
			if err != nil {
				log.Fatal("failed to handshake: ", err)
			}
			log.Printf("SSH: Connection from %s (%s)", conn.RemoteAddr(), conn.ClientVersion())

			// The incoming Request channel must be serviced.
			go ssh.DiscardRequests(reqs)
			go handleServer(conn.Permissions.Extensions["pubkey"], sorciaConf, chans)
		}()
	}
}

// RunSSH ...
func RunSSH(sorciaConf *setting.BaseStruct) {
	// Public key authentication is done by comparing
	// the public key of a received connection
	// with the entries in the authorized_keys file.
	authorizedKeysBytes, err := ioutil.ReadFile("/home/git/.ssh/authorized_keys")
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
		Config: ssh.Config{
			Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128"},
		},

		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	keyDir := "/home/git/.ssh"
	keyPath := filepath.Join(keyDir, "id_rsa")
	if _, err := os.Stat(keyPath); err != nil || !os.IsExist(err) {
		if err := os.MkdirAll(filepath.Dir(keyDir), os.ModePerm); err != nil {
			fmt.Println(err)
			return
		}

		cmd := exec.Command("ssh-keygen", "-f", keyPath, "-t", "rsa", "-m", "PEM", "-N", "")
		cmd.Dir = keyDir

		var out, stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
		}
		fmt.Printf("SSH: New private key is generateed: %s", keyPath)
	}

	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}
	config.AddHostKey(private)

	runSSH(config, sorciaConf, "0.0.0.0", "22")
}
