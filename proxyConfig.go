package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type proxyConfig struct {
	Domains     map[string]domainConfig `json:"domains" yaml:"domains"`
	Generic     string                  `json:"generic" yaml:"generic"`
	Docker      dockerConfig            `json:"docker" yaml:"docker"`
	ListenHTTP  string                  `json:"listenHTTP" yaml:"listenHTTP"`
	ListenHTTPS string                  `json:"listenHTTPS" yaml:"listenHTTPS"`
}

type domainConfig struct {
	SSL      sslConfig `json:"ssl,omitempty" yaml:"ssl,omitempty"`
	Slug     string    `json:"slug" yaml:"slug"`
	ForceSSL bool      `json:"force_ssl" yaml:"force_ssl"`
}

type sslConfig struct {
	Cert string `json:"cert" yaml:"cert"`
	Key  string `json:"key" yaml:"key"`
}

type dockerConfig struct {
	Hosts map[string]string `json:"hosts" yaml:"hosts"`
	Port  int               `json:"port" yaml:"port"`
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
