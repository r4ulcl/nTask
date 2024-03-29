package utils

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// CreateTLSClientWithCACert from cert.pem
func CreateTLSClientWithCACert(caCertPath string, verifyAltName, verbose, debug bool) (*http.Client, error) {

	tlsConfig, err := GenerateTLSConfig(caCertPath, verifyAltName, verbose, debug)
	if err != nil {
		if debug {
			log.Println("Utils Error reading worker config file: ", err)
		}
		return nil, err
	}

	// Create HTTP client with TLS
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return client, nil
}

func LoadWorkerConfig(filename string, verbose, debug bool) (*WorkerConfig, error) {
	var config WorkerConfig
	content, err := os.ReadFile(filename)
	if err != nil {
		if debug {
			log.Println("Utils Error reading worker config file: ", err)
		}
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		if debug {
			log.Println("Utils Error unmarshalling worker config: ", err)
		}
		return &config, err
	}

	// if Name is empty use hostname
	if config.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			if debug {
				log.Println("Utils Error getting hostname:", err)
			}
			return &config, err
		}
		if debug {
			log.Println("Utils hostname:", hostname)
		}
		config.Name = hostname
	}

	// Print the values from the struct
	if debug {
		log.Println("Utils Name:", config.Name)
		log.Println("Utils Tasks:")

		for module, exec := range config.Modules {
			log.Printf("  Module: %s, Exec: %s\n", module, exec)
		}
	}

	if verbose {
		log.Println("Config loaded:", config)
	}

	return &config, nil
}

func GenerateTLSConfig(caCertPath string, verifyAltName, verbose, debug bool) (*tls.Config, error) {
	var tlsConfig *tls.Config

	// Load CA certificate from file
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		fmt.Printf("Failed to read CA certificate file: %v\n", err)
		return nil, err
	}

	// Create a certificate pool and add the CA certificate
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	//var cert *x509.Certificate
	// Create a TLS configuration with the custom VerifyPeerCertificate function
	if !verifyAltName {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true, // Enable server verification
			RootCAs:            certPool,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				if len(rawCerts) == 0 {
					return fmt.Errorf("no certificates provided by the server")
				}

				serverCert, err := x509.ParseCertificate(rawCerts[0])
				if err != nil {
					return fmt.Errorf("failed to parse server certificate: %v", err)
				}

				// Verify the server certificate against the CA certificate
				opts := x509.VerifyOptions{
					Roots:         certPool,
					Intermediates: x509.NewCertPool(),
				}
				_, err = serverCert.Verify(opts)
				if err != nil {
					return fmt.Errorf("failed to verify server certificate: %v", err)
				}

				return nil
			},
		}
	} else {
		log.Println("Utils verifyAltName YES", !verifyAltName)

		tlsConfig = &tls.Config{
			InsecureSkipVerify: false, // Ensure that server verification is enabled
			RootCAs:            certPool,
			MinVersion:         tls.VersionTLS12, // Set the desired minimum TLS version
		}
	}

	return tlsConfig, nil
}
