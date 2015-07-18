package pulse

//Inspired by http://www.hydrogen18.com/blog/your-own-pki-tls-golang.html

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

func GeneratePrivKeyFile(fname string) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}
	blk := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(pk),
	}
	f, err := os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(pem.EncodeToMemory(blk))
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
}

func PrintCertRequest(privfname, reqfname string) string {
	log.Println(privfname)
	privraw, err := ioutil.ReadFile(privfname)
	if err != nil {
		log.Fatal(err)
	}
	blk, _ := pem.Decode(privraw)
	pk, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	template := &x509.CertificateRequest{
		Subject:            pkix.Name{CommonName: "Unnamed-Agent"}, //TODO Randomize maybe
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, template, pk)
	if err != nil {
		log.Fatal(err)
	}
	csrblk := &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csr,
	}

	f, err := os.Create(reqfname)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Fill and send following form to your Pulse host :-")
	outstr := "-------------- BEGIN EMAIL---------------------\n\n"
	outstr += "Agent Name       : ________________________________\n"
	outstr += "Country          : ________________________________\n"
	outstr += "State            : ________________________________\n"
	outstr += "City             : ________________________________\n"
	outstr += "ISP Resolver IPs : ________________________________\n\n"
	outstr += string(pem.EncodeToMemory(csrblk))
	outstr += "\n---------------- END EMAIL---------------------\n"
	fmt.Println(outstr)
	f.Write([]byte(outstr))
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	//log.Println(pk.N)
	data := fmt.Sprintf("Modulus=%X\n", pk.N.Bytes())
	//log.Println(data)
	return fmt.Sprintf("%x", sha1.Sum([]byte(data)))
}
