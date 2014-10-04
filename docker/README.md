# luzifer/dockerproxy Dockerfile

This repository contains **Dockerfile** of [Luzifer/dockerproxy](https://github.com/Luzifer/dockerproxy) for [Docker](https://www.docker.com/)'s [automated build](https://registry.hub.docker.com/u/luzifer/dockerproxy/) published to the public [Docker Hub Registry](https://registry.hub.docker.com/).

## Base Docker Image

- [golang](https://registry.hub.docker.com/_/golang/)

## Installation

1. Install [Docker](https://www.docker.com/).

2. Download [automated build](https://registry.hub.docker.com/u/luzifer/dockerproxy/) from public [Docker Hub Registry](https://registry.hub.docker.com/): `docker pull luzifer/php5-nginx`

## Usage

At first write your configuration as documented in the [dockerproxy readme](https://github.com/Luzifer/dockerproxy/blob/master/README.md#dockerproxy). For this container pay attention to use addresses `:80` and `:443` or change the ports below. If you specify SSL certificates they should have absolute paths. In this example a useful path would be `/etc/dockerproxy/ssl/yourcert.crt`.

Create a `Dockerfile` similar to the following in your configuration folder: 

```
FROM luzifer/dockerproxy

RUN mkdir -p /etc/dockerproxy
ADD . /etc/dockerproxy

EXPOSE 80
EXPOSE 443
CMD ["/go/bin/dockerproxy", "-configfile=/etc/dockerproxy/config.json"]
```

As an alternative you can use a docker container who has a mountpoint for the configuration and reads it from the host:

```
FROM luzifer/dockerproxy

VOLUME ["/etc/dockerproxy"]

EXPOSE 80
EXPOSE 443
CMD ["/go/bin/dockerproxy", "-configfile=/etc/dockerproxy/config.json"]
```

Then, execute the following to build the image:

```
docker build -t myuser/dockerproxy .
```

This will create an image named `myuser/dockerproxy` with your configuration ready to go.

To launch it, just type:

```
docker run -d -p 80 -p 443 myuser/dockerproxy
```

If you used the option to have a mount point you have to mount the configuration:

```
docker run -d -p 80 -p 443 -v /home/myuser/config:/etc/dockerproxy myuser/dockerproxy
```

Easy!

