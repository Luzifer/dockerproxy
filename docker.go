package main

import (
	"fmt"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

type dockerContainers map[string][]string

func collectDockerContainer() *dockerContainers {
	result := make(dockerContainers)

	for dockerHostPrivate, dockerHost := range proxyConfiguration.Docker.Hosts {
		// Connect every docker host and get its containers
		endpoint := fmt.Sprintf("tcp://%s:%d", dockerHostPrivate, proxyConfiguration.Docker.Port)
		client, _ := docker.NewClient(endpoint)
		containers, _ := client.ListContainers(docker.ListContainersOptions{})

		for _, apiContainer := range containers {
			container, _ := client.InspectContainer(apiContainer.ID)

			// Load slug and port from container labels
			if routerSlug, ok := container.Config.Labels["io.luzifer.dockerproxy.slug"]; ok {
				routerPort := container.Config.Labels["io.luzifer.dockerproxy.port"]
				result[routerSlug] = append(result[routerSlug], fmt.Sprintf("%s:%s", dockerHost, routerPort))
				continue // If new configuration is present don't parse env configuration
			}

			// Load ROUTER_SLUG and ROUTER_PORT from environment configuration of that container
			currentEnv := make(map[string]string)
			for _, envVar := range container.Config.Env {
				t := strings.Split(envVar, "=")
				currentEnv[t[0]] = t[1]
			}
			if slug, ok := currentEnv["ROUTER_SLUG"]; ok {
				port := currentEnv["ROUTER_PORT"]
				result[slug] = append(result[slug], fmt.Sprintf("%s:%s", dockerHost, port))
			}
		}
	}

	return &result
}
