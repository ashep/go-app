package buflogwriter_test

import (
	"testing"

	"github.com/ashep/go-app/buflogwriter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufLogWriter_Write(t *testing.T) {
	w := buflogwriter.New()
	n, err := w.Write([]byte("aString"))

	require.NoError(t, err)
	assert.Equal(t, 7, n)
}

func TestBufLogWriter_String(t *testing.T) {
	w := buflogwriter.New()

	_, err := w.Write([]byte("theString"))
	require.NoError(t, err)

	assert.Equal(t, "theString", w.String())

	// Second read returns the same
	assert.Equal(t, "theString", w.String())
}
