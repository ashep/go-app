package httplogwriter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ashep/go-apprun/httplogwriter"
)

func TestWriter_New(tt *testing.T) {
	tt.Run("EmptyURL", func(t *testing.T) {
		_, err := httplogwriter.New("", "", "", nil)
		assert.EqualError(tt, err, "empty url")
	})

	tt.Run("InvalidURL", func(t *testing.T) {
		_, err := httplogwriter.New(string([]byte{0x0}), "", "", nil)
		assert.EqualError(tt, err, `invalid url: parse "\x00": net/url: invalid control character in URL`)
	})

	tt.Run("NilHTTPClient", func(t *testing.T) {
		_, err := httplogwriter.New("anURL", "", "", nil)
		assert.NoError(tt, err)
	})

	tt.Run("Ok", func(t *testing.T) {
		_, err := httplogwriter.New("anURL", "", "", nil)
		assert.NoError(tt, err)
	})
}
