package sshtunnel

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/r4ulcl/nTask/manager/utils"
	"golang.org/x/crypto/ssh"
)

func forwardData(src, dest net.Conn) {
	_, err := io.Copy(src, dest)
	if err != nil {
		log.Printf("Error forwarding data: %v", err)
	}

	src.Close()
	dest.Close()
}

func publicKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

func StartSSH(config *utils.ManagerSSHConfig, portAPI string, verbose, debug bool) {
	log.Println("SSH StartSSH")
	for {

		for ip, port := range config.IPPort {
			go func(ip, port string) {
				log.Println("SSH connecction", ip, port)

				if !checkFileExists(config.PrivateKeyPath) {
					log.Fatal("File ", config.PrivateKeyPath, " not found")
				}

				auth, err := publicKeyFile(config.PrivateKeyPath)
				if err != nil {
					log.Fatal("Error loading file ", config.PrivateKeyPath, err)
				}

				// SSH connection configuration
				sshConfig := &ssh.ClientConfig{
					User: config.SSHUsername,
					Auth: []ssh.AuthMethod{
						auth,
					},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				}

				// If a password is provided, add it as an additional authentication method
				if config.PrivateKeyPassword != "" {
					sshConfig.Auth = append(sshConfig.Auth, ssh.Password(config.PrivateKeyPassword))
				}

				// Connect to the SSH server
				sshClient, err := ssh.Dial("tcp", ip+":"+port, sshConfig)
				if err != nil {
					log.Printf("Failed to dial: %s", err)
					return
				}

				// Remote port to forward
				remoteAddr := "127.0.0.1:" + portAPI
				// Local address to forward to
				localAddr := "127.0.0.1:" + portAPI

				if debug {
					log.Println("SSH remoteAddr", remoteAddr)
				}

				// Request remote port forwarding
				remoteListener, err := sshClient.Listen("tcp", remoteAddr)
				if err != nil {
					log.Printf("Failed to request remote port forwarding: %v", err)
					return
				}
				defer remoteListener.Close()

				fmt.Printf("Remote port forwarding %s to %s via SSH...\n", remoteAddr, localAddr)

				for {
					// Wait for a connection on the remote port
					remoteConn, err := remoteListener.Accept()
					if err != nil {
						log.Printf("Failed to accept connection on remote port: %v", err)
						return
					}

					// Connect to the local server
					localConn, err := net.Dial("tcp", localAddr)
					if err != nil {
						log.Printf("Failed to connect to local server: %v", err)
						remoteConn.Close()
						continue
					}

					// Start forwarding data between local and remote connections
					go forwardData(remoteConn, localConn)
					go forwardData(localConn, remoteConn)
				}
			}(ip, port)
		}
		time.Sleep(time.Second * 60)
	}
}

// ssh-keygen -t rsa -b 2048
func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(err, os.ErrNotExist)
}
