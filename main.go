package main

import (
	"io"
	"log"
	"net/http"

	"github.com/Luzifer/dockerproxy/sni"
	"github.com/Luzifer/rconfig"
	"github.com/hydrogen18/stoppableListener"
	"github.com/robfig/cron"
)

var (
	cfg = struct {
		ConfigFile        string `flag:"configfile" default:"./config.json" description:"Location of the configuration file"`
		LetsEncryptServer string `flag:"letsencrypt-server" default:"https://acme-v01.api.letsencrypt.org/directory" description:"ACME directory endpoint"`
	}{}

	containers         *dockerContainers
	proxyConfiguration *proxyConfig
	leClient           *letsEncryptClient
	sniServer          = sni.SNIServer{}
)

func init() {
	var err error

	if err := rconfig.Parse(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline flags: %s", err)
	}

	proxyConfiguration, err = newProxyConfig(cfg.ConfigFile)
	if err != nil {
		log.Fatalf("Unable to parse configuration: %s", err)
	}

	leClient, err = newLetsEncryptClient(cfg.LetsEncryptServer)
	if err != nil {
		log.Fatalf("Unable to create LetsEncrypt client: %s", err)
	}
}

func startSSLServer(proxy *dockerProxy, serverErrorChan chan error) {
	// Collect certificates from disk
	certificates := proxy.getCertificates()

	// Get a certificate for all LetsEncrypt enabled domains
	leDomains := []string{}
	for domain, domainCFG := range proxyConfiguration.Domains {
		if domainCFG.UseLetsEncrypt {
			leDomains = append(leDomains, domain)
		}
	}

	cert, key, err := leClient.FetchMultiDomainCertificate(leDomains)
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

		serverErrorChan <- sniServer.ListenAndServeTLSSNI(httpsServer, certificates)
	}(proxy, certificates)
}

func startHTTPServer(proxy *dockerProxy, serverErrorChan chan error) {
	letsEncryptHandler := http.HandlerFunc(func(res http.ResponseWriter, r *http.Request) {
		if challenge, ok := leClient.Challenges[r.Host]; ok {
			if r.URL.RequestURI() == challenge.Path {
				log.Printf("Got challenge request for domain %s and answered.", r.Host)
				io.WriteString(res, challenge.Response)
				return
			}
		}

		proxy.ServeHTTP(res, r)
	})

	go func(h http.Handler) {
		serverErrorChan <- http.ListenAndServe(proxyConfiguration.ListenHTTP, h)
	}(letsEncryptHandler)
}

func reloadConfiguration() {
	tmp, err := newProxyConfig(cfg.ConfigFile)
	if err == nil {
		proxyConfiguration = tmp
	} else {
		log.Printf("%v\n", err)
	}
	containers = collectDockerContainer()
}

func main() {
	containers = collectDockerContainer()
	proxy := newDockerProxy()

	c := cron.New()
	c.AddFunc("@every 1m", reloadConfiguration)
	c.AddFunc("@every 720h", func() {
		// Stop the SNI server every 30d, it will get restarted and
		// the LetsEncrypt certificates are checked for expiry
		sniServer.Stop()
	})
	c.Start()

	serverErrorChan := make(chan error, 2)

	startHTTPServer(proxy, serverErrorChan)
	startSSLServer(proxy, serverErrorChan)

	for {
		select {
		case err := <-serverErrorChan:
			if err != stoppableListener.StoppedError {
				log.Fatal(err)
			} else {
				// Something made the SNI server stop, we just restart it
				startSSLServer(proxy, serverErrorChan)
			}
		}
	}
}
