package main

import (
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Luzifer/dockerproxy/sni"
	"github.com/elazarl/goproxy"

	"github.com/Luzifer/dockerproxy/auth"
	_ "github.com/Luzifer/dockerproxy/auth/basic"
)

type DockerProxy struct {
	proxy *goproxy.ProxyHttpServer
}

func NewDockerProxy() *DockerProxy {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysReject)

	// We are not really a proxy but act as a HTTP(s) server who delivers remote pages
	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(w, req)
	})

	rand.Seed(time.Now().UnixNano())

	return &DockerProxy{
		proxy: proxy,
	}
}

func (d *DockerProxy) ServeHTTP(res http.ResponseWriter, r *http.Request) {
	d.shieldOwnHosts(d.httpLog(d.proxy)).ServeHTTP(res, r)
}

func (d *DockerProxy) GetCertificates() []sni.Certificates {
	var certs []sni.Certificates
	for _, domain := range proxyConfiguration.Domains {
		if domain.SSL.Cert != "" {
			certs = append(certs, sni.Certificates{
				CertFile: domain.SSL.Cert,
				KeyFile:  domain.SSL.Key,
			})
		}
	}
	return certs
}

func (d *DockerProxy) normalizeRemoteAddr(remote_addr string) string {
	idx := strings.LastIndex(remote_addr, ":")
	if idx != -1 {
		remote_addr = remote_addr[0:idx]
		if remote_addr[0] == '[' && remote_addr[len(remote_addr)-1] == ']' {
			remote_addr = remote_addr[1 : len(remote_addr)-1]
		}
	}
	return remote_addr
}

func (d *DockerProxy) httpLog(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s %s", d.normalizeRemoteAddr(r.RemoteAddr), r.Method, r.Host, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func (d *DockerProxy) shieldOwnHosts(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		slug := ""
		// Host is defined and slug has been found
		if host, ok := proxyConfiguration.Domains[req.Host]; ok {
			slug = host.Slug

			if host.ForceSSL && req.TLS == nil {
				req.URL.Scheme = "https"
				req.URL.Host = req.Host
				http.Redirect(w, req, req.URL.String(), 301)
				return
			}

			if proxyConfiguration.Domains[req.Host].Authentication.Type != "" {
				authHandler, err := auth.GetAuthHandler(proxyConfiguration.Domains[req.Host].Authentication.Type)
				if err != nil {
					http.Error(w, "Authentication system is misconfigured for this host.", http.StatusInternalServerError)
					log.Printf("AuthSystemError: %s\n", err)
					return
				}

				ok, err := authHandler(proxyConfiguration.Domains[req.Host].Authentication.Config, w, req)
				if err != nil {
					http.Error(w, "Authentication system threw an error.", http.StatusInternalServerError)
					log.Printf("AuthSystemError: %s\n", err)
					return
				}

				if !ok {
					http.Error(w, "Unauthorized.", http.StatusUnauthorized)
					return
				}
			}
		}
		// Host is a generic host
		if strings.HasSuffix(req.Host, proxyConfiguration.Generic) {
			slug = strings.Replace(req.Host, proxyConfiguration.Generic, "", -1)
		}
		// We found a valid slug before?
		if target, ok := (*containers)[slug]; ok && slug != "" {
			req.URL.Scheme = "http"
			req.URL.Host = target[rand.Intn(len(target))]
			req.Header.Add("X-Forwarded-For", d.normalizeRemoteAddr(req.RemoteAddr))

			handler.ServeHTTP(w, req)
		} else {
			http.Error(w, "This host is currently not available", 503)
		}
	})
}
