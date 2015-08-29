package auth // import "github.com/Luzifer/dockerproxy/auth"

import (
	"fmt"
	"net/http"

	"gopkg.in/yaml.v2"
)

type AuthFunction func(interface{}, http.ResponseWriter, *http.Request) (bool, error)

var authHandlers map[string]AuthFunction

func init() {
	authHandlers = make(map[string]AuthFunction)
}

func RegisterAuthHandler(name string, fn AuthFunction) {
	if _, existing := authHandlers[name]; existing {
		panic(fmt.Sprintf("AuthHandler with name '%s' already exisists", name))
	}
	authHandlers[name] = fn
}

func GetAuthHandler(name string) (AuthFunction, error) {
	if fn, ok := authHandlers[name]; ok {
		return fn, nil
	}
	return nil, fmt.Errorf("Unable to find authentication type '%s'", name)
}

func RemapConfiguration(i interface{}, o interface{}) error {
	tmp, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(tmp, o)
	return err
}
