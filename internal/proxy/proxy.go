package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// RequestLog captures metadata about a proxied HTTP request.
type RequestLog struct {
	ID         string        `json:"id"` // Optional: could be useful for correlation
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`
}

// ProxyHandler wraps the reverse proxy and captures request logs.
type ProxyHandler struct {
	proxy   *httputil.ReverseProxy
	logChan chan<- RequestLog
}

// NewProxy creates a new reverse proxy that forwards requests to targetURL
// and emits request logs to the provided channel.
func NewProxy(targetURL string, logChan chan<- RequestLog) (*ProxyHandler, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// Optional: Custom ErrorHandler to capture proxy errors (e.g., target down)
	originalErrorHandler := proxy.ErrorHandler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		if originalErrorHandler != nil {
			originalErrorHandler(w, r, err)
		} else {
			w.WriteHeader(http.StatusBadGateway)
		}
	}

	return &ProxyHandler{
		proxy:   proxy,
		logChan: logChan,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap the ResponseWriter to capture the status code
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	// Forward the request
	h.proxy.ServeHTTP(sw, r)

	duration := time.Since(start)

	// Create and emit the log
	reqLog := RequestLog{
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: sw.status,
		Duration:   duration,
		Timestamp:  start,
	}

	// Non-blocking send to avoid holding up the request if the consumer is slow
	select {
	case h.logChan <- reqLog:
	default:
		// Channel is full or no one is listening; drop the log or handle accordingly
		// log.Println("Warning: RequestLog channel full, dropping log")
	}
}

// statusWriter is a wrapper around http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
