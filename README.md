# Luzifer / dockerproxy

DockerProxy is a small application to shield HTTP exporting [Docker](https://www.docker.com/) containers. The proxy supports SNI to shield the containers with HTTPs certificates. To discover the containers the Docker daemon needs to listen on a tcp port which should be shielded by a firewall to ensure the security of the Docker host.

## Design Flaw

Currently Docker does not support container tagging so this proxy is using the environment variables to detect the "slug" and the port of a container. This can be fixed as soon as there is a tagging concept similar as the EC2 tagging in AWS.

## Configuration

### Docker daemon

- Ensure the daemon is [listening on a tcp port](https://docs.docker.com/articles/basics/#bind-docker-to-another-hostport-or-a-unix-socket) reachable from the dockerproxy. In this example port `9999` is used.
- Start your docker containers with some special environment variables used for container detection:
  - `ROUTER_SLUG`: The slug used in the proxy configuration to identify the container
  - `ROUTER_PORT`: The public exported HTTP port the proxy can send its requests to

### dockerproxy

The configuration is written in JSON format and read every minute by the daemon:

- `domains`: Dict of domain configurations the proxy is able to respond to
  - `slug`: The slug defined in the Docker container to determine which container should handle the request
  - `force_ssl`: The proxy does not forward request but return a redirect to SSL based connection
  - `ssl` (optional): SSL configuration for that domain
    - `cert`: x509 certificate file (Intermediate certificates belongs in this file too. Put them under your own certificate.)
    - `key`: The key for the cerficate without password protection
- `generic`: A generic suffix on which the proxy will forward to every configured container
- `listenHTTP`: An address binding for HTTP traffic like `:80`
- `listenHTTPS`: An address binding for HTTPs traffic like `:443`
- `docker`: Docker host configuration
  - `hosts`: Dict of private to public host/ip associations (The Proxy will query the Docker daemon on the private host/ip and send traffic to the public host/ip)
  - `port`: Port to use for querying the Docker daemon

Example configuration:

```
{
  "domains": {
    "host1.example.com": {
      "ssl": {
        "cert": "ssl/host1.example.com.crt",
        "key": "ssl/host1.example.com.key"
      },
      "slug": "container1",
      "force_ssl": true
    },
    "host2.example.com": {
      "slug": "container2"
    }
  },
  "generic": ".dockersrv.example.com",
  "listenHTTP": ":8081",
  "listenHTTPS": ":4443",
  "docker": {
    "hosts": {
      "localhost": "docker01.servers.example.com"
    },
    "port": 9999
  }
}
```

