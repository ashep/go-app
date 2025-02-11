package cfgloader

import (
	"fmt"
	"strings"
)

func LoadFromPath(path string, out interface{}, schema []byte) error {
	switch {
	case strings.HasSuffix(path, ".yml"), strings.HasSuffix(path, ".yaml"):
		return LoadYAMLFromPath(path, out, schema)
	case strings.HasSuffix(path, ".json"):
		return LoadJSONFromPath(path, out, schema)
	}

	return fmt.Errorf("cannot determine file format: %s", path)
}
