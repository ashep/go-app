package httplogger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ashep/go-apprun/httplogger"
	"github.com/ashep/go-apprun/testlogger"
)

func TestWrap(tt *testing.T) {
	tt.Run("EmptyURL", func(t *testing.T) {
		l, _ := testlogger.New()

		_, err := httplogger.New(l, "", "", "", nil)
		assert.EqualError(tt, err, "empty url")
	})

	tt.Run("InvalidURL", func(t *testing.T) {
		l, _ := testlogger.New()

		_, err := httplogger.New(l, string([]byte{0x0}), "", "", nil)
		assert.EqualError(tt, err, `invalid url: parse "\x00": net/url: invalid control character in URL`)
	})

	tt.Run("NilHTTPClient", func(t *testing.T) {
		l, _ := testlogger.New()
		_, err := httplogger.New(l, "anURL", "", "", nil)
		assert.NoError(tt, err)
	})

	tt.Run("Ok", func(t *testing.T) {
		l, _ := testlogger.New()
		_, err := httplogger.New(l, "anURL", "", "", nil)
		assert.NoError(tt, err)
	})
}
