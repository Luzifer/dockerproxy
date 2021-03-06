package main

//go:generate make bindata

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Luzifer/dockerproxy/sni"
	"github.com/Luzifer/go_helpers/str"
	"github.com/Luzifer/rconfig"
	"github.com/gorilla/mux"
	"github.com/hydrogen18/stoppableListener"
	"github.com/prometheus/client_golang/prometheus"
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

	requestCount    *prometheus.CounterVec
	requestDuration prometheus.Summary
	responseSize    prometheus.Summary
)

func initMetrics() {
	so := prometheus.SummaryOpts{
		Subsystem:   "http",
		ConstLabels: prometheus.Labels{"handler": "dockerproxy"},
	}

	reqCnt := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem:   so.Subsystem,
		Name:        "requests_total",
		Help:        "Total number of HTTP requests made.",
		ConstLabels: so.ConstLabels,
	}, []string{"method", "code"})

	so.Name = "response_size_bytes"
	so.Help = "The HTTP response sizes in bytes."
	resSz := prometheus.NewSummary(so)

	so.Name = "request_duration_microseconds"
	so.Help = "The HTTP request latencies in microseconds."
	reqDur := prometheus.NewSummary(so)

	requestCount = prometheus.MustRegisterOrGet(reqCnt).(*prometheus.CounterVec)
	requestDuration = prometheus.MustRegisterOrGet(reqDur).(prometheus.Summary)
	responseSize = prometheus.MustRegisterOrGet(resSz).(prometheus.Summary)
}

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

	initMetrics()
}

func createDomainMap(domains []string) map[string][]string {
	result := make(map[string][]string)

	for _, domain := range domains {
		parts := strings.Split(domain, ".")
		if len(parts) < 2 {
			// This isn't a domain
			continue
		}

		secondLevel := strings.Join(parts[len(parts)-2:], ".")
		if _, ok := result[secondLevel]; !ok {
			if str.StringInSlice(secondLevel, domains) {
				result[secondLevel] = []string{secondLevel}
			} else {
				result[secondLevel] = []string{domain}
			}
		}
		result[secondLevel] = str.AppendIfMissing(result[secondLevel], domain)
	}

	return result
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

	domainMap := createDomainMap(leDomains)

	if len(domainMap) > 0 {
		for _, leDomains := range domainMap {
			cert, key, err := leClient.FetchMultiDomainCertificate(leDomains)
			if err != nil {
				log.Fatalf("ERROR: Unable to get certificate: %s", err)
			}
			intermediate, err := leClient.GetIntermediateCertificate()
			if err != nil {
				log.Fatalf("ERROR: Unable to get intermediate certificate: %s", err)
			}
			certificates = append(certificates, sni.Certificates{
				Certificate:  cert,
				Key:          key,
				Intermediate: intermediate,
			})
		}
	}

	go func(proxy http.Handler, certificates []sni.Certificates) {
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
				requestCount.WithLabelValues("acme", "200").Inc()
				io.WriteString(res, challenge.Response)
				return
			}
		}

		// If we see unanswered acme challenges reject them instead redirecting them to the application
		if strings.Contains(r.URL.RequestURI(), ".well-known/acme-challenge") {
			requestCount.WithLabelValues("acme", "404").Inc()
			http.Error(res, "Invalid acme-challenge", http.StatusNotFound)
			return
		}

		proxy.ServeHTTP(res, r)
	})

	go func(h http.Handler) {
		serverErrorChan <- http.ListenAndServe(proxyConfiguration.ListenHTTP, h)
	}(letsEncryptHandler)
}

func startMetricsServer(serverErrorChan chan error) {
	r := mux.NewRouter()
	r.Handle("/metrics", prometheus.Handler())

	go func(h http.Handler) {
		serverErrorChan <- http.ListenAndServe(proxyConfiguration.ListenMetrics, h)
	}(r)
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
	startMetricsServer(serverErrorChan)

	for err := range serverErrorChan {
		if err != stoppableListener.StoppedError {
			log.Fatal(err)
		} else {
			// Something made the SNI server stop, we just restart it
			startSSLServer(proxy, serverErrorChan)
		}
	}
}
