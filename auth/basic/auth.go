package basic // import "github.com/Luzifer/dockerproxy/auth/basic"

import (
	"fmt"
	"net/http"

	"github.com/Luzifer/dockerproxy/auth"
)

func init() {
	auth.RegisterAuthHandler("basic-auth", CheckBasicAuth)
}

type basicAuthConfig map[string]string

func CheckBasicAuth(config interface{}, res http.ResponseWriter, r *http.Request) (bool, error) {
	cfg := make(basicAuthConfig)
	err := auth.RemapConfiguration(config, &cfg)
	if err != nil {
		return false, err
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		res.Header().Add("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", r.URL.Host))
		return false, nil
	}

	if pwd, ok := cfg[username]; ok && pwd == password {
		return true, nil
	}

	res.Header().Add("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", r.URL.Host))
	return false, nil
}
