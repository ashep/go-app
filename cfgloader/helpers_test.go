package cfgloader_test

import (
	"os"
	"path"
	"testing"
)

type cfgStruct struct {
	Foo   string  `yaml:"foo" json:"foo"`
	Bar   string  `yaml:"bar" json:"bar"`
	Int   int     `yaml:"int" json:"int"`
	Float float64 `yaml:"float" json:"float"`
	Baz   struct {
		Foo   string  `yaml:"foo" json:"foo"`
		Bar   string  `yaml:"bar" json:"bar"`
		Int   int     `yaml:"int" json:"int"`
		Float float64 `yaml:"float" json:"float"`
	} `yaml:"baz" json:"baz"`
}

var testSchema = []byte(`{
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"type": "object",
	"properties": {
		"foo": {
			"type": "string",
			"maxLength": 2
		},
		"bar": {
			"type": "string",
			"minLength": 1
		},
		"baz": {
			"type": "object",
			"properties": {
				"foo": {
					"type": "string"
				},
				"bar": {
					"type": "string"
				}
			}
		}
	}
}
`)

func writeTempFile(t *testing.T, b []byte, ext string) (string, error) {
	p := path.Join(t.TempDir(), "testFile"+ext)

	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // ok

	if _, err = f.Write(b); err != nil {
		return "", err
	}

	return p, nil
}
