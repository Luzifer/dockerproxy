build-linux:
		docker run --rm -ti \
				-w /go/src/github.com/Luzifer/dockerproxy \
				-v $(CURDIR):/go/src/github.com/Luzifer/dockerproxy \
				golang go build .

bindata:
		go-bindata assets
