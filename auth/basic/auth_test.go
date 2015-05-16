package basic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestAuth(t *testing.T) {
	yamlString := "---\nconfig:\n  alice: CheshireCat\n  bob: goobar"
	cfg := struct {
		Config interface{}
	}{}
	err := yaml.Unmarshal([]byte(yamlString), &cfg)
	if err != nil {
		t.Fatalf("Unable to decode inline yaml: %s", err)
	}

	right := map[string]string{
		"alice": "CheshireCat",
		"bob":   "goobar",
	}
	wrong := map[string]string{
		"alice": "foobar",
		"knut":  "test",
		"bob":   "CheshireCat",
	}

	for user, pass := range right {
		r, _ := http.NewRequest("GET", "/", nil)
		r.SetBasicAuth(user, pass)
		res := httptest.NewRecorder()
		ok, err := CheckBasicAuth(cfg.Config, res, r)
		if err != nil {
			t.Errorf("An error is present: %s", err)
		}
		if !ok {
			t.Errorf("User/Password was rejected: %s:%s", user, pass)
		}
		if res.Header().Get("WWW-Authenticate") != "" {
			t.Errorf("WWW-Authenticate was found")
		}
	}

	for user, pass := range wrong {
		r, _ := http.NewRequest("GET", "/", nil)
		r.SetBasicAuth(user, pass)
		res := httptest.NewRecorder()
		ok, err := CheckBasicAuth(cfg.Config, res, r)
		if err != nil {
			t.Errorf("An error is present: %s", err)
		}
		if ok {
			t.Errorf("User/Password was accepted: %s:%s", user, pass)
		}
		if res.Header().Get("WWW-Authenticate") == "" {
			t.Errorf("WWW-Authenticate was not found")
		}
	}
}
