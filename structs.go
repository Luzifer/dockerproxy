package main

type config struct {
	Domains     map[string]domainConfig `json:"domains"`
	Generic     string                  `json:"generic"`
	Docker      dockerConfig            `json:"docker"`
	ListenHTTP  string                  `json:"listenHTTP"`
	ListenHTTPS string                  `json:"listenHTTPS"`
}

type domainConfig struct {
	SSL      sslConfig `json:"ssl,omitempty"`
	Slug     string    `json:"slug"`
	ForceSSL bool      `json:"force_ssl"`
}

type sslConfig struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

type dockerConfig struct {
	Hosts map[string]string `json:"hosts"`
	Port  int               `json:"port"`
}
