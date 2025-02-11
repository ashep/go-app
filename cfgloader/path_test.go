package cfgloader_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ashep/go-cfgloader"
)

func TestLoadFromFile(tt *testing.T) {
	tt.Run("JSONAsYAML", func(t *testing.T) {
		p, err := writeTempFile(t, []byte(`{"foo":"ba","bar":"baz"}`), ".yaml")
		require.NoError(t, err)

		out := cfgStruct{}
		err = cfgloader.LoadFromPath(p, &out, testSchema)

		require.NoError(t, err)
		assert.Equal(t, "ba", out.Foo)
		assert.Equal(t, "baz", out.Bar)
	})

	tt.Run("YAMLAsJSON", func(t *testing.T) {
		p, err := writeTempFile(t, []byte("foo: ba\nbar: baz"), ".json")
		require.NoError(t, err)

		out := cfgStruct{}
		err = cfgloader.LoadFromPath(p, &out, testSchema)

		require.EqualError(t, err, "invalid character 'o' in literal false (expecting 'a')")
		assert.NotErrorIs(t, err, cfgloader.SchemaValidationError{})
	})

	tt.Run("OkJSON", func(t *testing.T) {
		p, err := writeTempFile(t, []byte(`{"foo":"ba","bar":"baz"}`), ".json")
		require.NoError(t, err)

		out := cfgStruct{}
		err = cfgloader.LoadFromPath(p, &out, testSchema)

		require.NoError(t, err)
		assert.Equal(t, "ba", out.Foo)
		assert.Equal(t, "baz", out.Bar)
	})

	tt.Run("OkYAML", func(t *testing.T) {
		p, err := writeTempFile(t, []byte("foo: ba\nbar: baz"), ".yaml")
		require.NoError(t, err)

		out := cfgStruct{}
		err = cfgloader.LoadFromPath(p, &out, testSchema)

		require.NoError(t, err)
		assert.Equal(t, "ba", out.Foo)
		assert.Equal(t, "baz", out.Bar)
	})
}
