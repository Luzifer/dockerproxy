package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Luzifer/dockerproxy/sni"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/elazarl/goproxy.v1"
)

func loadConfig() config {
	file, e := ioutil.ReadFile("./config.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}
	var cfg config
	err := json.Unmarshal(file, &cfg)
	if err != nil {
		fmt.Printf("JSON error: %v\n", err)
		os.Exit(1)

	}
	return cfg
}

func collectDockerContainer(cfg *config) map[string]string {
	result := make(map[string]string)
	for dockerHostPrivate, dockerHost := range cfg.Docker.Hosts {
		endpoint := fmt.Sprintf("tcp://%s:%d", dockerHostPrivate, cfg.Docker.Port)
		client, _ := docker.NewClient(endpoint)
		containers, _ := client.ListContainers(docker.ListContainersOptions{})
		for _, apiContainer := range containers {
			container, _ := client.InspectContainer(apiContainer.ID)
			currentEnv := make(map[string]string)
			for _, envVar := range container.Config.Env {
				var k, v string
				unpack(strings.Split(envVar, "="), &k, &v)
				currentEnv[k] = v
			}
			if slug, ok := currentEnv["ROUTER_SLUG"]; ok {
				port := currentEnv["ROUTER_PORT"]
				result[slug] = fmt.Sprintf("%s:%s", dockerHost, port)
			}
		}
	}

	return result
}

func unpack(s []string, vars ...*string) {
	for i, str := range s {
		*vars[i] = str
	}
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	cfg := loadConfig()
	containers := collectDockerContainer(&cfg)

	// We are not really a proxy but act as a HTTP(s) server who delivers remote pages
	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		slug := ""
		// Host is defined and slug has been found
		if host, ok := cfg.Domains[req.Host]; ok {
			slug = host.Slug
		}
		// Host is a generic host
		if strings.HasSuffix(req.Host, cfg.Generic) {
			slug = strings.Replace(req.Host, cfg.Generic, "", -1)
		}
		// We found a valid slug before?
		if target, ok := containers[slug]; ok {
			req.URL.Scheme = "http"
			req.URL.Host = target

			proxy.ServeHTTP(w, req)
		} else {
			http.Error(w, "This host is currently not available", 502)
		}
	})

	var certs []sni.Certificates
	for _, domain := range cfg.Domains {
		if domain.SSL.Cert != "" {
			certs = append(certs, sni.Certificates{
				CertFile: domain.SSL.Cert,
				KeyFile:  domain.SSL.Key,
			})
		}
	}

	httpChan := make(chan error)
	httpsChan := make(chan error)
	loaderChan := time.NewTicker(time.Minute).C

	go func(proxy *goproxy.ProxyHttpServer) {
		httpChan <- http.ListenAndServe(cfg.ListenHTTP, proxy)
	}(proxy)

	go func(*goproxy.ProxyHttpServer) {
		httpsServer := &http.Server{
			Handler: proxy,
			Addr:    cfg.ListenHTTPS,
		}

		httpsChan <- sni.ListenAndServeTLSSNI(httpsServer, certs)
	}(proxy)

	for {
		select {
		case httpErr := <-httpChan:
			log.Fatal(httpErr)
		case httpsErr := <-httpsChan:
			log.Fatal(httpsErr)
		case <-loaderChan:
			cfg = loadConfig()
			containers = collectDockerContainer(&cfg)
		}
	}
}
