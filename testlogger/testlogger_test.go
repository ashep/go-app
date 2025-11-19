package testlogger_test

import (
	"testing"

	"github.com/ashep/go-app/testlogger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogger(main *testing.T) {
	main.Parallel()

	main.Run("Log", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := zerolog.New(tl.Logger())

		l.Info().Msg("test message")
		assert.Equal(t, `{"level":"info","message":"test message"}
`, tl.Content())
	})
}
