build-linux:
	docker run --rm -ti \
		-w /go/src/github.com/Luzifer/dockerproxy \
		-v $(CURDIR):/go/src/github.com/Luzifer/dockerproxy \
		golang go build .

bindata:
	go-bindata assets

ci:
	curl -sSLo golang.sh https://raw.githubusercontent.com/Luzifer/github-publish/master/golang.sh
	bash golang.sh
