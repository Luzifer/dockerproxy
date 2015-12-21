package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"

	"github.com/ericchiang/letsencrypt"
)

var supportedChallenges = []string{
	letsencrypt.ChallengeHTTP,
}

func requestLetsEncryptCertificate(domains []string) (*x509.Certificate, *rsa.PrivateKey, error) {
	cli, err := letsencrypt.NewClient(cfg.LetsEncryptServer)
	if err != nil {
		return nil, nil, err
	}

	accountKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	if _, err := cli.NewRegistration(accountKey); err != nil {
		return nil, nil, err
	}

	for _, domain := range domains {
		// ask for a set of challenges for a given domain
		auth, _, err := cli.NewAuthorization(accountKey, "dns", domain)
		if err != nil {
			return nil, nil, err
		}
		chals := auth.Combinations(supportedChallenges...)
		if len(chals) == 0 {
			return nil, nil, fmt.Errorf("no supported challenge combinations")
		}

		// HTTP Challenge handling
		chal := chals[0][0]
		if chal.Type != letsencrypt.ChallengeHTTP {
			return nil, nil, fmt.Errorf("Did not find a HTTP challenge")
		}

		path, resource, err := chal.HTTP(accountKey)
		if err != nil {
			return nil, nil, err
		}

		letsEncryptChallenges[domain] = struct {
			Path     string
			Response string
		}{
			Path:     path,
			Response: resource,
		}

		// Tell the server the challenge is ready and poll the server for updates.
		if err := cli.ChallengeReady(accountKey, chal); err != nil {
			// oh no, you failed the challenge
			return nil, nil, err
		}
		// The challenge has been verified!
	}

	csr, certKey, err := newCSR(domains)
	if err != nil {
		return nil, nil, err
	}

	// Request a certificate for your domain
	cert, err := cli.NewCertificate(accountKey, csr)
	return cert, certKey, err

}

func newCSR(domains []string) (*x509.CertificateRequest, *rsa.PrivateKey, error) {
	certKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	template := &x509.CertificateRequest{
		SignatureAlgorithm: x509.SHA256WithRSA,
		PublicKeyAlgorithm: x509.RSA,
		PublicKey:          &certKey.PublicKey,
		Subject:            pkix.Name{CommonName: domains[0]},
		DNSNames:           domains,
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, certKey)
	if err != nil {
		return nil, nil, err
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, nil, err
	}
	return csr, certKey, nil
}
