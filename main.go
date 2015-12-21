package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Luzifer/dockerproxy/sni"
	"github.com/Luzifer/rconfig"
)

var (
	cfg = struct {
		ConfigFile        string `flag:"configfile" default:"./config.json" description:"Location of the configuration file"`
		LetsEncryptServer string `flag:"letsencrypt-server" default:"https://acme-v01.api.letsencrypt.org/directory" description:"ACME directory endpoint"`
	}{}
	letsEncryptChallenges = map[string]struct {
		Path     string
		Response string
	}{}

	containers         *dockerContainers
	proxyConfiguration *proxyConfig
)

func init() {
	var err error

	if err := rconfig.Parse(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline flags: %s", err)
	}

	proxyConfiguration, err = newProxyConfig(cfg.ConfigFile)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func main() {
	containers = collectDockerContainer()
	proxy := newDockerProxy()

	serverErrorChan := make(chan error, 2)
	loaderChan := time.NewTicker(time.Minute)

	letsEncryptHandler := http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		if challenge, ok := letsEncryptChallenges[r.Host]; ok {
			if r.URL.RequestURI() == challenge.Path {
				io.WriteString(res, challenge.Response)
				return
			}
		}

		proxy.ServeHTTP(res, r)
	})

	go func(h http.Handler) {
		serverErrorChan <- http.ListenAndServe(proxyConfiguration.ListenHTTP, h)
	}(letsEncryptHandler)

	// Collect certificates from disk
	certificates := proxy.getCertificates()

	// Get a certificate for all LetsEncrypt enabled domains
	leDomains := []string{}
	for domain, domainCFG := range proxyConfiguration.Domains {
		if domainCFG.UseLetsEncrypt {
			leDomains = append(leDomains, domain)
		}
	}

	cert, key, err := requestLetsEncryptCertificate(leDomains)
	if err != nil {
		log.Fatalf("ERROR: Unable to get certificate: %s", err)
	}
	certificates = append(certificates, sni.Certificates{
		Certificate: cert,
		Key:         key,
	})

	go func(proxy *dockerProxy, certificates []sni.Certificates) {
		httpsServer := &http.Server{
			Handler: proxy,
			Addr:    proxyConfiguration.ListenHTTPS,
		}

		serverErrorChan <- sni.ListenAndServeTLSSNI(httpsServer, certificates)
	}(proxy, certificates)

	for {
		select {
		case err := <-serverErrorChan:
			log.Fatal(err)
		case <-loaderChan.C:
			tmp, err := newProxyConfig(cfg.ConfigFile)
			if err == nil {
				proxyConfiguration = tmp
			} else {
				log.Printf("%v\n", err)
			}
			containers = collectDockerContainer()
		}
	}
}
