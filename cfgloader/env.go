package cfgloader

import (
	"github.com/kelseyhightower/envconfig"
)

func LoadFromEnv(prefix string, out interface{}) error {
	return envconfig.Process(prefix, out)
}
