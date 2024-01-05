package sshTunnel

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

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

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := os.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func StartSSH(config *utils.ManagerSSHConfig, portAPI string, verbose, debug bool) {
	//return nil
	log.Println("StartSSH")
	for ip, port := range config.IpPort {
		log.Println("--------------", ip, port)

		// SSH connection configuration
		sshConfig := &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				// Use the private key for authentication if provided
				publicKeyFile(config.PrivateKeyPath),
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
			log.Fatalf("Failed to dial: %s", err)
		}

		// Remote port to forward
		remoteAddr := "127.0.0.1:" + portAPI
		// Local address to forward to
		localAddr := "127.0.0.1:" + portAPI

		if debug {
			log.Println("remoteAddr", remoteAddr)
		}

		// Request remote port forwarding
		remoteListener, err := sshClient.Listen("tcp", remoteAddr)
		if err != nil {
			log.Fatalf("Failed to request remote port forwarding: %v", err)
		}
		defer remoteListener.Close()

		fmt.Printf("Remote port forwarding %s to %s via SSH...\n", remoteAddr, localAddr)

		for {
			// Wait for a connection on the remote port
			remoteConn, err := remoteListener.Accept()
			if err != nil {
				log.Fatalf("Failed to accept connection on remote port: %v", err)
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
	}
}

// ssh-keygen -t rsa -b 2048
