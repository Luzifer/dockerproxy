package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ericchiang/letsencrypt"
	homedir "github.com/mitchellh/go-homedir"
)

var (
	supportedChallenges = []string{
		letsencrypt.ChallengeHTTP,
	}
)

const (
	renewTimeLeft = 30 * 24 * time.Hour
)

func init() {
	gob.Register(letsEncryptClientCache{})
	gob.Register(rsa.PrivateKey{})
	gob.Register(rsa.PublicKey{})
	gob.Register(x509.Certificate{})
}

type letsEncryptClientCache struct {
	AccountKey   *rsa.PrivateKey
	Certificates map[string]letsEncryptClientCertificateCache
}

type letsEncryptClientCertificateCache struct {
	Certificate *x509.Certificate
	Key         *rsa.PrivateKey
}

type letsEncryptClientChallenge struct {
	Path     string
	Response string
}

type letsEncryptClientChallenges map[string]letsEncryptClientChallenge

type letsEncryptClient struct {
	Challenges letsEncryptClientChallenges

	client *letsencrypt.Client

	// Caching
	cache     letsEncryptClientCache
	cacheFile string
}

func newLetsEncryptClient(server string) (*letsEncryptClient, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	homedir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	cacheFile := path.Join(homedir, ".config", "dockerproxy.lecache")
	os.MkdirAll(path.Dir(cacheFile), 0600)

	cache := letsEncryptClientCache{
		Certificates: make(map[string]letsEncryptClientCertificateCache),
	}
	if _, err := os.Stat(cacheFile); err == nil {
		cf, err := os.Open(cacheFile)
		if err != nil {
			return nil, err
		}
		defer cf.Close()
		if err := gob.NewDecoder(cf).Decode(&cache); err != nil {
			return nil, err
		}
	}

	cli, err := letsencrypt.NewClient(server)
	if err != nil {
		return nil, err
	}

	return &letsEncryptClient{
		Challenges: make(letsEncryptClientChallenges),

		client:    cli,
		cache:     cache,
		cacheFile: cacheFile,
	}, nil
}

func (l *letsEncryptClient) saveCache() error {
	os.MkdirAll(path.Dir(l.cacheFile), 0600)

	cf, err := os.Create(l.cacheFile)
	if err != nil {
		return err
	}
	defer cf.Close()

	return gob.NewEncoder(cf).Encode(l.cache)
}

func (l *letsEncryptClient) log(format string, args ...interface{}) {
	log.Printf("[LetsEncrypt] "+format, args...)
}

func (l *letsEncryptClient) getAccountKey() (*rsa.PrivateKey, error) {
	if l.cache.AccountKey != nil {
		return l.cache.AccountKey, nil
	}

	l.log("Registering new AccountKey")

	accountKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	if _, err := l.client.NewRegistration(accountKey); err != nil {
		return nil, err
	}

	l.cache.AccountKey = accountKey
	if err := l.saveCache(); err != nil {
		return nil, err
	}

	return accountKey, nil
}

func (l *letsEncryptClient) authorizeDomain(domain string) error {
	log.Printf("Authorizing domain: %s", domain)

	accountKey, err := l.getAccountKey()
	if err != nil {
		return err
	}

	// ask for a set of challenges for a given domain
	auth, _, err := l.client.NewAuthorization(accountKey, "dns", domain)
	if err != nil {
		return err
	}
	chals := auth.Combinations(supportedChallenges...)
	if len(chals) == 0 {
		return fmt.Errorf("no supported challenge combinations")
	}

	// HTTP Challenge handling
	chal := chals[0][0]
	if chal.Type != letsencrypt.ChallengeHTTP {
		return fmt.Errorf("Did not find a HTTP challenge")
	}

	path, resource, err := chal.HTTP(accountKey)
	if err != nil {
		return err
	}

	l.Challenges[domain] = letsEncryptClientChallenge{
		Path:     path,
		Response: resource,
	}

	// Tell the server the challenge is ready and poll the server for updates.
	return l.client.ChallengeReady(accountKey, chal)
}

func (l *letsEncryptClient) hashMultiDomain(domains []string) string {
	sort.Strings(domains)
	rawString := strings.Join(domains, "::")
	return fmt.Sprintf("%x", sha1.Sum([]byte(rawString)))
}

func (l *letsEncryptClient) FetchMultiDomainCertificate(domains []string) (*x509.Certificate, *rsa.PrivateKey, error) {
	domainHash := l.hashMultiDomain(domains)
	if cert, ok := l.cache.Certificates[domainHash]; ok && cert.Certificate.NotAfter.Sub(time.Now()) > renewTimeLeft {
		log.Printf("Using cached certificate for domains %s", strings.Join(domains, ", "))
		return cert.Certificate, cert.Key, nil
	}

	accountKey, err := l.getAccountKey()
	if err != nil {
		return nil, nil, err
	}

	for _, domain := range domains {
		if err := leClient.authorizeDomain(domain); err != nil {
			return nil, nil, err
		}
	}

	csr, certKey, err := l.createMultiDomainCSR(domains)
	if err != nil {
		return nil, nil, err
	}

	cert, err := l.client.NewCertificate(accountKey, csr)
	if err != nil {
		return nil, nil, err
	}

	l.cache.Certificates[domainHash] = letsEncryptClientCertificateCache{
		Certificate: cert,
		Key:         certKey,
	}
	if err := l.saveCache(); err != nil {
		return nil, nil, err
	}

	log.Printf("Fetched fresh certificate for domains %s", strings.Join(domains, ", "))
	return cert, certKey, err
}

func (l *letsEncryptClient) createMultiDomainCSR(domains []string) (*x509.CertificateRequest, *rsa.PrivateKey, error) {
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

func (l *letsEncryptClient) GetIntermediateCertificate() (*x509.Certificate, error) {
	crtData, err := Asset("assets/lets-encrypt-x3-cross-signed.pem")
	if err != nil {
		return nil, err
	}
	b, _ := pem.Decode(crtData)
	return x509.ParseCertificate(b.Bytes)
}
