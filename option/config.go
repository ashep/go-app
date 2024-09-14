package option

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ashep/go-cfgloader"

	"github.com/ashep/go-apprun/runner"
)

func WithConfigFromFiles[CT any](appName string) Option[CT] {
	return func(_ context.Context, r *runner.Runner[CT]) error {
		// Try to load from "standard" paths
		for _, base := range []string{"config", appName} {
			for _, ext := range []string{".yaml", ".json"} {
				cfgPath := base + ext
				err := cfgloader.LoadFromPath(cfgPath, &r.Config, nil)
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("config load failed: %w", err)
				}

				r.Logger.Debug().Str("path", cfgPath).Msg("config loaded from file")
			}
		}

		// From a path defined by an env variable
		if cfgPath := os.Getenv("APP_CONFIG_PATH"); cfgPath != "" {
			if err := cfgloader.LoadFromPath(cfgPath, &r.Config, nil); err != nil {
				return fmt.Errorf("config load failed: %w", err)
			}

			r.Logger.Debug().Str("path", cfgPath).Msg("config loaded from file")
		}

		return nil
	}
}

func WithConfigFromEnv[CT any]() Option[CT] {
	return func(_ context.Context, r *runner.Runner[CT]) error {
		if err := cfgloader.LoadFromEnv("APP", &r.Config); err != nil {
			return fmt.Errorf("load config from env vars failed: %w", err)
		}

		r.Logger.Debug().Msg("config loaded from env vars")

		return nil
	}
}
