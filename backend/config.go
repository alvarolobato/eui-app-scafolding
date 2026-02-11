package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type appConfig struct {
	AdminSecret string `yaml:"admin_secret"`

	// EncryptionKeys holds an optional list of base64-encoded keys
	// used for encrypting and signing secrets, such as credentials
	// and OAuth state cookies. Before base64-encoding, the keys
	// should be either 32 or 64 random bytes.
	//
	// The first entry in EncryptionKeys will be used for encoding
	// new values, while any entry may be used for decoding,
	// enabling key rotation.
	EncryptionKeys []string `yaml:"encryption_keys"`

	Elasticsearch struct {
		URL    string `yaml:"url"`
		APIKey string `yaml:"api_key"`
	} `yaml:"elasticsearch"`

	Google struct {
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	} `yaml:"google"`
}

func setConfigFromEnv(cfg *appConfig) {
	var walk func(v reflect.Value, prefix string)
	walk = func(v reflect.Value, prefix string) {
		typ := v.Type()
		n := v.NumField()
		for i := 0; i < n; i++ {
			field := v.Field(i)
			yamlTag := typ.Field(i).Tag.Get("yaml")
			name := strings.ToUpper(prefix + yamlTag)
			switch field.Kind() {
			case reflect.Struct:
				walk(field, name+"_")
			case reflect.String:
				if v := os.Getenv(name); v != "" {
					field.Set(reflect.ValueOf(v))
				}
			case reflect.Slice:
				if v := os.Getenv(name); v != "" {
					field.Set(reflect.ValueOf(strings.Fields(v)))
				}
			default:
				panic(fmt.Sprintf("%s: %s", name, typ))
			}
		}
	}
	walk(reflect.ValueOf(cfg).Elem(), "")
}

func loadConfig(path string) (*appConfig, error) {
	var cfg appConfig
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
			return nil, err
		}
	}
	setConfigFromEnv(&cfg)
	return &cfg, nil
}
