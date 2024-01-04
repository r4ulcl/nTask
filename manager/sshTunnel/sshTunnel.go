package sshTunnel

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"

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
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func StartSSH(config *utils.ManagerSSHConfig, verbose, debug bool) {
	//return nil
	log.Println("StartSSH")
	for ip, port := range config.IpPort {
		log.Println("--------------", ip, port)

		// SSH connection configuration
		sshConfig := &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				// Use the private key for authentication
				publicKeyFile(config.PrivateKeyPath),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		// Connect to the SSH server
		sshClient, err := ssh.Dial("tcp", ip+":"+port, sshConfig)
		if err != nil {
			log.Fatalf("Failed to dial: %s", err)
		}

		// Remote port to forward
		remoteAddr := "127.0.0.1:8180"
		// Local address to forward to
		localAddr := "127.0.0.1:8180"

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
