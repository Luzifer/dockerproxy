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

	"github.com/hydrogen18/stoppableListener"
)

// Certificates is a representation of a certificate and a key
type Certificates struct {
	CertFile string
	KeyFile  string

	Certificate  *x509.Certificate
	Key          *rsa.PrivateKey
	Intermediate *x509.Certificate
}

type SNIServer struct {
	listener *stoppableListener.StoppableListener
}

func (s *SNIServer) Stop() {
	s.listener.Stop()
}

// ListenAndServeTLSSNI openes a http listener with SNI certificate selection
// from the Certificates collection
func (s *SNIServer) ListenAndServeTLSSNI(srv *http.Server, certs []Certificates) error {
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
			certPEM := pem.EncodeToMemory(&pem.Block{Bytes: v.Certificate.Raw, Type: "CERTIFICATE"})
			if v.Intermediate != nil {
				certPEM = append(certPEM, '\n')
				certPEM = append(certPEM, pem.EncodeToMemory(&pem.Block{Bytes: v.Intermediate.Raw, Type: "CERTIFICATE"})...)
			}

			config.Certificates[i], err = tls.X509KeyPair(
				certPEM,
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

	// ++++ SSL security settings

	// Force clients to use TLS1.0 as SSL is buggy as hell
	config.MinVersion = tls.VersionTLS10

	// Force TLS negotiation to pick a result from the (presumably) stronger list of server-specified ciphers
	config.PreferServerCipherSuites = true

	// Pick secure cipher suites
	config.CipherSuites = []uint16{
		// 256bit
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		// 128bit
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		// 112bit
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	}

	// ++++ End SSL security settings

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.listener, err = stoppableListener.New(conn)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(s.listener, config)
	return srv.Serve(tlsListener)
}
