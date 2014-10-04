FROM golang

MAINTAINER Knut Ahlers <knut@ahlers.me>

RUN go get github.com/Luzifer/dockerproxy
RUN go install github.com/Luzifer/dockerproxy

ENTRYPOINT /go/bin/dockerproxy
