package sni

import (
	"crypto/tls"
	"net"
	"net/http"
)

// Certificates is a representation of a certificate and a key
type Certificates struct {
	CertFile string
	KeyFile  string
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
		config.Certificates[i], err = tls.LoadX509KeyPair(v.CertFile, v.KeyFile)
		if err != nil {
			return err
		}
	}

	config.BuildNameToCertificate()

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(conn, config)
	return srv.Serve(tlsListener)
}
