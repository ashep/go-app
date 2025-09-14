package health_test

import (
	"net/http"
	"testing"

	"github.com/ashep/go-app/health"
	"github.com/ashep/go-app/testhttpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterServer(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		s := testhttpserver.New(t)
		health.RegisterServer(s)
		s.Run()

		res, err := http.Get(s.URL(health.URLPath))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}
