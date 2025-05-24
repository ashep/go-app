package cfgloader

import (
	"encoding/json"
	"io"
	"os"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

func LoadYAML(in []byte, out interface{}, schema []byte) error {
	if err := yaml.Unmarshal(in, out); err != nil {
		return err
	}

	if schema != nil {
		outJSON, err := json.Marshal(out)
		if err != nil {
			return err
		}

		res, err := gojsonschema.Validate(
			gojsonschema.NewBytesLoader(schema),
			gojsonschema.NewBytesLoader(outJSON),
		)
		if err != nil {
			return err
		}
		if !res.Valid() {
			return SchemaValidationError{Result: res}
		}
	}

	return nil
}

func LoadYAMLFromPath(path string, out interface{}, schema []byte) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // ok

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	return LoadYAML(b, out, schema)
}
