#!/bin/bash

# Set the Common Name for the Certificate Authority
CA_NAME="nTask"
O_NAME="r4ulcl"

# Set the Common Names for the SSL certificates
MANAGER_CERT_NAME="Manager"

# Set folder names for each server
MANAGER_FOLDER="manager"

# Set IP and hostname information
MANAGER_IP="192.168.1.10"
MANAGER_HOSTNAME="manager.local"

# Create directories to store the CA and certificate files
mkdir -p certs/${MANAGER_FOLDER}

# Step 1: Generate a private key for the Certificate Authority (CA)
openssl genpkey -algorithm RSA -out certs/ca-key.pem

# Step 2: Generate a self-signed certificate for the CA
openssl req -x509 -new -key certs/ca-key.pem -out certs/ca-cert.pem -subj "/CN=${CA_NAME}/O=${O_NAME}"

# Copy the CA certificate to each server folder
cp certs/ca-cert.pem certs/${MANAGER_FOLDER}/

# Step 3: Generate a private key for the Manager SSL certificate
openssl genpkey -algorithm RSA -out certs/${MANAGER_FOLDER}/key.pem

# Step 4: Generate a Certificate Signing Request (CSR) for the Manager SSL certificate
openssl req -new -key certs/${MANAGER_FOLDER}/key.pem -out certs/${MANAGER_FOLDER}/csr.pem -subj "/CN=${MANAGER_CERT_NAME}/O=${O_NAME}" -addext "subjectAltName = IP:${MANAGER_IP},DNS:${MANAGER_HOSTNAME}"

# Step 5: Sign the Manager SSL certificate with the CA
openssl x509 -req -in certs/${MANAGER_FOLDER}/csr.pem -CA certs/ca-cert.pem -CAkey certs/ca-key.pem -out certs/${MANAGER_FOLDER}/cert.pem -CAcreateserial -extfile <(printf "subjectAltName = IP:${MANAGER_IP},DNS:${MANAGER_HOSTNAME}") -days 365

# Optional: Display information about the generated certificates
echo "Manager Certificate:"
openssl x509 -in certs/${MANAGER_FOLDER}/cert.pem -noout -text

echo "Certificates and CA generated successfully. Files are located in the 'certs' directory."
