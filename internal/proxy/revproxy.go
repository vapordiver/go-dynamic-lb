package revproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	StatusClientClosed = 499
)

var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

type RevProxy struct {
	transport http.RoundTripper
}

func New(transport http.RoundTripper) *RevProxy {
	proxy := &RevProxy{
		transport: transport,
	}

	return proxy
}

func NewDefault() *RevProxy {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = 5 * time.Second

	return New(transport)
}

func (rp *RevProxy) Forward(w http.ResponseWriter, r *http.Request, target *url.URL) (int, error) {
	outReq := r.Clone(r.Context())

	outReq.URL.Scheme = target.Scheme
	outReq.URL.Host = target.Host
	outReq.Host = target.Host

	outReq.RequestURI = ""
	outReq.Close = false

	cleanHopByHop(outReq.Header)
	outReq.Header.Add("X-Forwarded-Host", r.Host)
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		forwardedIPs := strings.Join(outReq.Header.Values("X-Forwarded-For"), ", ")
		if forwardedIPs != "" {
			clientIP = forwardedIPs + ", " + clientIP
		}
		outReq.Header.Set("X-Forwarded-For", clientIP)
	}

	resp, err := rp.transport.RoundTrip(outReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(outReq.Context().Err(), context.Canceled) {
			return StatusClientClosed, err
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			// 504 Gateway Timeout
			w.WriteHeader(http.StatusGatewayTimeout)
			return http.StatusGatewayTimeout, err
		}
		// 502 Bad Gateway, etc.
		w.WriteHeader(http.StatusBadGateway)
		return http.StatusBadGateway, err

	}

	defer resp.Body.Close()

	cleanHopByHop(resp.Header)
	for header, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		// in case streaming body to response fails, return an adequate error
		return resp.StatusCode, fmt.Errorf("(RevProxy Forward) failed to stream body back to response: %w", err)
	}

	return resp.StatusCode, nil
}

func cleanHopByHop(h http.Header) {
	var tokens []string
	for _, line := range h.Values("Connection") {
		vals := strings.Split(line, ",")
		for _, val := range vals {
			if connHeader := strings.TrimSpace(val); connHeader != "" {
				tokens = append(tokens, connHeader)
			}
		}
	}

	for _, token := range tokens {
		h.Del(token)
	}

	for _, hopHeader := range hopHeaders {
		h.Del(hopHeader)
	}

}
