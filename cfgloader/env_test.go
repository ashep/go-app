package cfgloader_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ashep/go-cfgloader"
)

func TestLoadFromEnv(tt *testing.T) {
	tt.Run("Ok", func(t *testing.T) {
		_ = os.Setenv("APP_FOO", "foo")
		_ = os.Setenv("APP_BAR", "bar")
		_ = os.Setenv("APP_INT", "123")
		_ = os.Setenv("APP_FLOAT", "123.456")
		_ = os.Setenv("APP_BAZ_FOO", "baz foo")
		_ = os.Setenv("APP_BAZ_BAR", "baz bar")
		_ = os.Setenv("APP_BAZ_INT", "234")
		_ = os.Setenv("APP_BAZ_FLOAT", "234.567")

		cfg := cfgStruct{}
		require.NoError(t, cfgloader.LoadFromEnv("APP", &cfg))

		assert.Equal(t, "foo", cfg.Foo)
		assert.Equal(t, "bar", cfg.Bar)
		assert.Equal(t, 123, cfg.Int)
		assert.Equal(t, 123.456, cfg.Float)
		assert.Equal(t, "baz foo", cfg.Baz.Foo)
		assert.Equal(t, "baz bar", cfg.Baz.Bar)
		assert.Equal(t, 234, cfg.Baz.Int)
		assert.Equal(t, 234.567, cfg.Baz.Float)
	})
}
