package cfgloader

import (
	"github.com/kelseyhightower/envconfig"
)

func LoadFromEnv(prefix string, out any) error {
	return envconfig.Process(prefix, out)
}
