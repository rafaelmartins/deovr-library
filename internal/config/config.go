package config

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BaseURL string `yaml:"baseurl"`
	Addr    string `yaml:"addr"`
	Scenes  Scenes `yaml:"scenes"`
}

func Load(file string) (*Config, error) {
	rv := &Config{}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(rv); err != nil {
		return nil, err
	}

	if rv.BaseURL == "" {
		hn, err := os.Hostname()
		if err != nil {
			hn = "localhost"
		}
		rv.BaseURL = "http://" + hn
	}

	u, err := url.Parse(rv.BaseURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" || u.Hostname() == "" {
		return nil, errors.New("baseurl must be absolute")
	}

	if rv.Addr == "" {
		p := u.Port()
		if p == "" {
			if s := strings.ToLower(u.Scheme); s == "https" {
				p = "443"
			} else {
				p = "80"
			}
		}
		rv.Addr = ":" + p
	}

	return rv, nil
}
