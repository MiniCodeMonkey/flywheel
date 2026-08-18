package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIBase       string `yaml:"api_base"`
	Email         string `yaml:"email"`
	AppPublicKey  string `yaml:"app_public_key"`
	AppPrivateKey string `yaml:"app_private_key"`
	ClientVersion string `yaml:"client_version"`
}

func defaults() Config {
	return Config{
		APIBase:       "https://api.mowl.com",
		AppPublicKey:  "C00869F2-4457-486D-ADAD-47409605B187",
		AppPrivateKey: "8285D895-29F6-425B-AE60-1725A68FF42E",
		ClientVersion: "8.8.2",
	}
}

func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "flywheel"), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "flywheel"), nil
}

func Load() (Config, error) {
	c := defaults()
	dir, err := Dir()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	d := defaults() // fill any empty fields
	if c.APIBase == "" {
		c.APIBase = d.APIBase
	}
	if c.AppPublicKey == "" {
		c.AppPublicKey = d.AppPublicKey
	}
	if c.AppPrivateKey == "" {
		c.AppPrivateKey = d.AppPrivateKey
	}
	if c.ClientVersion == "" {
		c.ClientVersion = d.ClientVersion
	}
	return c, nil
}

func (c Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.yaml"), b, 0o600)
}

func tokenPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token"), nil
}

func SaveToken(tok string) error {
	p, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(tok), 0o600)
}

func LoadToken() (string, error) {
	p, err := tokenPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
