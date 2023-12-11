openssl genpkey -algorithm RSA -out cert/key.pem
openssl req -new -key cert/key.pem -out cert/csr.pem
openssl x509 -req -in cert/csr.pem -signkey cert/key.pem -out cert/cert.pem
