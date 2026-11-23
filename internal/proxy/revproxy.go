package revproxy

import (
	"net/http"
	"net/url"
	"time"
)

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

	proxy := &RevProxy{
		transport: transport,
	}

	return proxy
}

func (rp *RevProxy) Forward(w http.Response, r *http.Request, target *url.URL) (int, error) {
	outReq := r.Clone(r.Context())

	outReq.URL.Scheme = target.Scheme
	outReq.URL.Host = target.Host

	outReq.RequestURI = ""
	outReq.Close = false

	// clean hop-by-hop
	// X-Forwarded-For ...

	resp, err := rp.transport.RoundTrip(outReq)
	if err != nil {
		// do something
	}

	defer resp.Body.Close()

	// clean hop-by-hop
	// Add() each header from resp to writer
	// set status code
	// copy stream of response to writer

	return resp.StatusCode, err
}
