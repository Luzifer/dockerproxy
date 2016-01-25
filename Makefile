build-linux:
		docker run --rm -ti -w /go/src/github.com/Luzifer/dockerproxy -v $(CURDIR):/go/src/github.com/Luzifer/dockerproxy -e GOPATH=/go/src/github.com/Luzifer/dockerproxy/Godeps/_workspace:/go golang go build .

bindata:
		go-bindata assets
