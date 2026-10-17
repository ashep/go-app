package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient returns a client whose pauses between attempts are recorded instead of slept.
func newTestClient(t *testing.T, maxTries int) (*Client, *[]time.Duration) {
	t.Helper()

	var pauses []time.Duration
	c := New(zerolog.Nop())
	c.SetMaxTries(maxTries)
	c.sleep = func(ctx context.Context, d time.Duration) error {
		pauses = append(pauses, d)
		return ctx.Err()
	}

	return c, &pauses
}

// newTestServer answers with the given statuses in turn, repeating the last one, and counts requests.
func newTestServer(t *testing.T, statuses ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		i := int(n.Add(1)) - 1
		w.WriteHeader(statuses[min(i, len(statuses)-1)])
		_, _ = w.Write([]byte("body"))
	}))
	t.Cleanup(srv.Close)

	return srv, &n
}

func TestClient_DoRequest_FirstTry(t *testing.T) {
	srv, n := newTestServer(t, http.StatusOK)
	c, pauses := newTestClient(t, 3)

	res, body, err := c.DoRequest(context.Background(), http.MethodGet, srv.URL, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "body", string(body))
	assert.EqualValues(t, 1, n.Load())
	assert.Empty(t, *pauses)
}

func TestClient_DoRequest_RetriesServerError(t *testing.T) {
	srv, n := newTestServer(t, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusOK)
	c, pauses := newTestClient(t, 3)

	res, body, err := c.DoRequest(context.Background(), http.MethodGet, srv.URL, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "body", string(body))
	assert.EqualValues(t, 3, n.Load())
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, *pauses)
}

func TestClient_DoRequest_AllServerErrors(t *testing.T) {
	srv, n := newTestServer(t, http.StatusServiceUnavailable)
	c, pauses := newTestClient(t, 3)

	res, body, err := c.DoRequest(context.Background(), http.MethodGet, srv.URL, nil, nil)
	require.EqualError(t, err, "request failed after 3 tries: HTTP response status: 503 Service Unavailable")
	assert.Nil(t, res)
	assert.Nil(t, body)
	assert.EqualValues(t, 3, n.Load())
	assert.Len(t, *pauses, 2, "no pause after the last attempt")
}

func TestClient_DoRequest_AllTransportErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	c, _ := newTestClient(t, 2)

	res, body, err := c.DoRequest(context.Background(), http.MethodGet, srv.URL, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed after 2 tries")
	assert.Nil(t, res)
	assert.Nil(t, body)
	assert.EqualValues(t, 2, c.reqNum.Load())
}

func TestClient_DoRequest_ClientErrorIsNotRetried(t *testing.T) {
	srv, n := newTestServer(t, http.StatusNotFound)
	c, pauses := newTestClient(t, 3)

	res, body, err := c.DoRequest(context.Background(), http.MethodGet, srv.URL, nil, nil)
	require.EqualError(t, err, "HTTP response status: 404 Not Found")
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "body", string(body))
	assert.EqualValues(t, 1, n.Load())
	assert.Empty(t, *pauses)
}

func TestClient_DoRequest_SingleTry(t *testing.T) {
	for _, maxTries := range []int{1, 0, -1} {
		srv, n := newTestServer(t, http.StatusServiceUnavailable, http.StatusOK)
		c, pauses := newTestClient(t, maxTries)

		_, _, err := c.DoRequest(context.Background(), http.MethodGet, srv.URL, nil, nil)
		require.EqualError(t, err, "request failed after 1 tries: HTTP response status: 503 Service Unavailable",
			"maxTries=%d", maxTries)
		assert.EqualValues(t, 1, n.Load(), "maxTries=%d", maxTries)
		assert.Empty(t, *pauses, "maxTries=%d", maxTries)
	}
}

func TestClient_DoRequest_CanceledContext(t *testing.T) {
	srv, n := newTestServer(t, http.StatusOK)
	c, _ := newTestClient(t, 3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := c.DoRequest(ctx, http.MethodGet, srv.URL, nil, nil)
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, n.Load())
}

func TestClient_DoRequest_CanceledWhilePausing(t *testing.T) {
	srv, n := newTestServer(t, http.StatusServiceUnavailable)
	c := New(zerolog.Nop())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, err := c.DoRequest(ctx, http.MethodGet, srv.URL, nil, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), time.Second, "the 1s pause is cut short")
	assert.EqualValues(t, 1, n.Load())
}

func TestSleepCtx(t *testing.T) {
	require.NoError(t, sleepCtx(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, sleepCtx(ctx, time.Hour), context.Canceled)
}
