// Code by Paul van Brouwershaven
// https://groups.google.com/d/msg/golang-nuts/rUm2iYTdrU4/PaEBya4dzvoJ

package sni

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
)

// Certificates is a representation of a certificate and a key
type Certificates struct {
	CertFile string
	KeyFile  string

	Certificate *x509.Certificate
	Key         *rsa.PrivateKey
}

// ListenAndServeTLSSNI openes a http listener with SNI certificate selection
// from the Certificates collection
func ListenAndServeTLSSNI(srv *http.Server, certs []Certificates) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error

	config.Certificates = make([]tls.Certificate, len(certs))
	for i, v := range certs {
		if v.Certificate != nil {
			config.Certificates[i], err = tls.X509KeyPair(
				pem.EncodeToMemory(&pem.Block{Bytes: v.Certificate.Raw, Type: "CERTIFICATE"}),
				pem.EncodeToMemory(&pem.Block{Bytes: x509.MarshalPKCS1PrivateKey(v.Key), Type: "RSA PRIVATE KEY"}),
			)
			if err != nil {
				return err
			}
		} else {
			config.Certificates[i], err = tls.LoadX509KeyPair(v.CertFile, v.KeyFile)
			if err != nil {
				return err
			}
		}
	}

	config.BuildNameToCertificate()

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// Force clients to use TLS1.0 as SSL is buggy as hell
	config.MinVersion = tls.VersionTLS10

	tlsListener := tls.NewListener(conn, config)
	return srv.Serve(tlsListener)
}
