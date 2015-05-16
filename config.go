package main

import "flag"

type Config struct {
	ConfigFile string
}

func NewConfig() *Config {
	var (
		configFile = flag.String("configfile", "./config.json", "Location of the configuration file")
	)

	flag.Parse()

	return &Config{
		ConfigFile: *configFile,
	}
}
