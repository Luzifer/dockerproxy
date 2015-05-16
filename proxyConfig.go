package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type proxyConfig struct {
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

func newProxyConfig(configFile string) (*proxyConfig, error) {
	tmp := proxyConfig{}

	configBody, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to load config file: %s", err)
	}

	err = yaml.Unmarshal(configBody, &tmp)
	if err != nil {
		err := json.Unmarshal(configBody, &tmp)
		if err != nil {
			return nil, fmt.Errorf("Failed to read yaml & json from config file")
		}
	}

	return &tmp, nil
}
