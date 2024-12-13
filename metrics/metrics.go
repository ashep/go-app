package metrics

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	mux = sync.RWMutex{}

	counters   = make(map[string]*prometheus.CounterVec)
	histograms = make(map[string]*prometheus.HistogramVec)
)

func HTTPServerRequest(req *http.Request, labels prometheus.Labels) func(int) {
	lbs := prometheus.Labels{
		"method": req.Method,
		"host":   req.Host,
		"path":   req.URL.Path,
		"code":   "",
	}

	if labels != nil {
		for k, v := range labels {
			lbs[k] = v
		}
	}

	cnt := Counter("http_server_requests_total", "Total number of HTTP server requests", lbs)
	dur := Histogram("http_server_request_duration_seconds", "HTTP server request duration.", lbs)

	start := time.Now()
	return func(statusCode int) {
		lbs["code"] = strconv.Itoa(statusCode)
		cnt.With(lbs).Inc()
		dur.With(lbs).Observe(time.Since(start).Seconds())
	}
}

func Counter(name, help string, labels prometheus.Labels) *prometheus.CounterVec {
	k := metricKey(name, labels)

	mux.RLock()
	h, ok := counters[k]
	if !ok {
		mux.RUnlock()
		mux.Lock()
		defer mux.Unlock()

		h = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: name,
			Help: help,
		}, labelKeys(labels))

		counters[k] = h
	} else {
		mux.RUnlock()
	}

	return h
}

func Histogram(name, help string, labels prometheus.Labels) *prometheus.HistogramVec {
	k := metricKey(name, labels)

	mux.RLock()
	h, ok := histograms[k]
	if !ok {
		mux.RUnlock()
		mux.Lock()
		defer mux.Unlock()

		h = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: name,
			Help: help,
		}, labelKeys(labels))

		histograms[k] = h
	} else {
		mux.RUnlock()
	}

	return h
}

func labelKeys(labels prometheus.Labels) []string {
	res := make([]string, 0, len(labels))
	for k := range labels {
		res = append(res, k)
	}

	slices.Sort(res)

	return res
}

func metricKey(k string, labels prometheus.Labels) string {
	return k + strings.Join(labelKeys(labels), "")
}
