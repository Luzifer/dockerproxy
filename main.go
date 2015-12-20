package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Luzifer/dockerproxy/sni"
	"github.com/Luzifer/rconfig"
)

var (
	containers         *dockerContainers
	cfg                *config
	proxyConfiguration *proxyConfig
)

func init() {
	var err error

	cfg = struct {
		ConfigFile string `flag:"configfile" default:"./config.json" description:"Location of the configuration file"`
	}{}

	proxyConfiguration, err = newProxyConfig(cfg.ConfigFile)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func init() {
	if err := rconfig.Parse(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline flags: %s", err)
	}
}

func main() {
	containers = collectDockerContainer()
	proxy := newDockerProxy()

	serverErrorChan := make(chan error, 2)
	loaderChan := time.NewTicker(time.Minute)

	go func(proxy *dockerProxy) {
		serverErrorChan <- http.ListenAndServe(proxyConfiguration.ListenHTTP, proxy)
	}(proxy)

	go func(proxy *dockerProxy) {
		httpsServer := &http.Server{
			Handler: proxy,
			Addr:    proxyConfiguration.ListenHTTPS,
		}

		serverErrorChan <- sni.ListenAndServeTLSSNI(httpsServer, proxy.getCertificates())
	}(proxy)

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
