package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Luzifer/dockerproxy/sni"
	"github.com/Luzifer/go_helpers/accessLogger"
	"github.com/elazarl/goproxy"

	"github.com/Luzifer/dockerproxy/auth"
	_ "github.com/Luzifer/dockerproxy/auth/basic"
)

type dockerProxy struct {
	proxy *goproxy.ProxyHttpServer
}

type redirectRewriter struct{}

func (r redirectRewriter) HandleResp(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
	if resp == nil {
		return false
	}
	_, ok := resp.Header["Location"]
	return ok
}
func redirectRewriterRewrite(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	loc, _ := resp.Location()
	if host, ok := proxyConfiguration.Domains[loc.Host]; ok && host.ForceSSL && loc.Scheme == "http" {
		loc.Scheme = "https"
		resp.Header.Set("Location", loc.String())
	}
	return resp
}

func newDockerProxy() *dockerProxy {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysReject)

	proxy.OnResponse(redirectRewriter{}).DoFunc(redirectRewriterRewrite)

	// We are not really a proxy but act as a HTTP(s) server who delivers remote pages
	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(w, req)
	})

	rand.Seed(time.Now().UnixNano())

	return &dockerProxy{
		proxy: proxy,
	}
}

func (d *dockerProxy) ServeHTTP(res http.ResponseWriter, r *http.Request) {
	d.shieldOwnHosts(d.httpLog(d.proxy)).ServeHTTP(res, r)
}

func (d *dockerProxy) getCertificates() []sni.Certificates {
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

func (d *dockerProxy) normalizeRemoteAddr(remoteAddress string) string {
	idx := strings.LastIndex(remoteAddress, ":")
	if idx != -1 {
		remoteAddress = remoteAddress[0:idx]
		if remoteAddress[0] == '[' && remoteAddress[len(remoteAddress)-1] == ']' {
			remoteAddress = remoteAddress[1 : len(remoteAddress)-1]
		}
	}
	return remoteAddress
}

func (d *dockerProxy) httpLog(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		al := accessLogger.New(w)

		start := time.Now()
		handler.ServeHTTP(al, r)
		duration := float64(time.Since(start)) / float64(time.Microsecond)

		requestCount.WithLabelValues(
			strings.ToLower(r.Method),
			strconv.FormatInt(int64(al.StatusCode), 10),
		).Inc()
		responseSize.Observe(float64(al.Size))
		requestDuration.Observe(duration)

		log.Printf("%s %s \"%s %s\" %d %d \"%s\"",
			d.normalizeRemoteAddr(r.RemoteAddr),
			r.Host,
			r.Method, r.URL.RequestURI(),
			al.StatusCode, al.Size,
			r.Header.Get("User-Agent"),
		)
	})
}

func (d *dockerProxy) shieldOwnHosts(handler http.Handler) http.Handler {
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
