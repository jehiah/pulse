package pulse

//Inspired by http://www.hydrogen18.com/blog/your-own-pki-tls-golang.html

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
)

func loadCertificates(caFile, certificateFile, privateKeyFile string) (tls.Certificate, *x509.CertPool) {

	mycert, err := tls.LoadX509KeyPair(certificateFile, privateKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	pem, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatal(err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pem) {
		log.Fatal("Failed appending certs")
	}

	return mycert, certPool
}

func GetTLSConfig(caFile, certificateFile, privateKeyFile string) *tls.Config {
	config := &tls.Config{}
	mycert, certPool := loadCertificates(caFile, certificateFile, privateKeyFile)
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = mycert

	config.RootCAs = certPool
	config.ClientCAs = certPool

	config.ClientAuth = tls.RequireAndVerifyClientCert

	//Optional stuff

	//Use only modern ciphers
	config.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}

	//Use only TLS v1.2
	config.MinVersion = tls.VersionTLS12

	//Don't allow session resumption
	config.SessionTicketsDisabled = true
	return config
}
