package zerologhttpwriter_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ashep/go-apprun/testlogger"
	"github.com/ashep/go-apprun/zerologhttpwriter"
)

func TestWriter_New(tt *testing.T) {
	tt.Run("EmptyURL", func(t *testing.T) {
		l, _ := testlogger.New()

		_, err := zerologhttpwriter.New(l, "", nil)
		assert.EqualError(tt, err, "empty url")
	})

	tt.Run("InvalidURL", func(t *testing.T) {
		l, _ := testlogger.New()

		_, err := zerologhttpwriter.New(l, string([]byte{0x0}), nil)
		assert.EqualError(tt, err, `invalid url: parse "\x00": net/url: invalid control character in URL`)
	})

	tt.Run("NilHTTPClient", func(t *testing.T) {
		l, _ := testlogger.New()

		_, err := zerologhttpwriter.New(l, "anURL", nil)
		assert.EqualError(tt, err, "nil http client")
	})

	tt.Run("Ok", func(t *testing.T) {
		l, _ := testlogger.New()

		_, err := zerologhttpwriter.New(l, "anURL", &httpClientMock{})
		assert.NoError(tt, err)
	})
}

func TestWriter_Write(tt *testing.T) {
	tt.Run("HTTPRequestFailed", func(t *testing.T) {
		l, lb := testlogger.New()

		c := &httpClientMock{}
		defer c.AssertExpectations(t)

		c.On("Do", mock.Anything).
			Return((*http.Response)(nil), fmt.Errorf("theHTTPClientDoError"))

		w, err := zerologhttpwriter.New(l, "anURL", c)
		require.NoError(tt, err)

		_, err = w.Write([]byte("aMessage"))
		assert.EqualError(t, err, "could not send request: theHTTPClientDoError")

		assert.Equal(t, `{"message":"aMessage"}
`, lb.Content())
	})

	tt.Run("BadHTTPResponseStatus", func(t *testing.T) {
		l, lb := testlogger.New()

		c := &httpClientMock{}
		defer c.AssertExpectations(t)

		c.On("Do", mock.Anything).
			Return(&http.Response{StatusCode: http.StatusOK}, nil)

		w, err := zerologhttpwriter.New(l, "anURL", c)
		require.NoError(tt, err)

		_, err = w.Write([]byte("aMessage"))
		assert.EqualError(t, err, "invalid response status code: 200")

		assert.Equal(t, `{"message":"aMessage"}
`, lb.Content())
	})

	tt.Run("Ok", func(t *testing.T) {
		l, lb := testlogger.New()

		c := &httpClientMock{}
		defer c.AssertExpectations(t)

		c.On("Do", mock.AnythingOfType("*http.Request")).
			Run(func(args mock.Arguments) {
				req := args.Get(0).(*http.Request)
				assert.Equal(t, "theURL", req.URL.String())

				b, err := io.ReadAll(req.Body)
				require.NoError(tt, err)
				assert.Equal(t, []byte("theMessage"), b)
			}).
			Return(&http.Response{StatusCode: http.StatusCreated}, nil)

		w, err := zerologhttpwriter.New(l, "theURL", c)
		require.NoError(tt, err)

		_, err = w.Write([]byte("theMessage"))
		assert.NoError(t, err)

		assert.Equal(t, `{"message":"theMessage"}
`, lb.Content())
	})
}

type httpClientMock struct {
	mock.Mock
}

func (m *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}
