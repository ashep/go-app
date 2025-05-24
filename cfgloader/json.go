package cfgloader

import (
	"encoding/json"
	"io"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

func LoadJSON(in []byte, out interface{}, schema []byte) error {
	err := json.Unmarshal(in, out)
	if err != nil {
		return err
	}

	if schema != nil {
		res, err := gojsonschema.Validate(
			gojsonschema.NewBytesLoader(schema),
			gojsonschema.NewBytesLoader(in),
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

func LoadJSONFromPath(path string, out interface{}, schema []byte) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // ok

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	return LoadJSON(b, out, schema)
}
