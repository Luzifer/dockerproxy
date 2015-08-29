package main

import "flag"

type config struct {
	ConfigFile string
}

func newConfig() *config {
	var (
		configFile = flag.String("configfile", "./config.json", "Location of the configuration file")
	)

	flag.Parse()

	return &config{
		ConfigFile: *configFile,
	}
}
