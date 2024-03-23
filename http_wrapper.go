package sling

import (
	"io"
	"net/http"
)

type HttpWrapper struct {
	http *http.Client
}

func (h *HttpWrapper) Do(req *http.Request) (*http.Response, []byte, error) {
	resp, err := h.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	// when err is nil, resp contains a non-nil resp.Body which must be closed
	defer resp.Body.Close()

	// The default HTTP client's Transport may not
	// reuse HTTP/1.x "keep-alive" TCP connections if the Body is
	// not read to completion and closed.
	// See: https://golang.org/pkg/net/http/#Response
	defer io.Copy(io.Discard, resp.Body)
	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return resp, rawData, nil
}

func NewHttpWrapper(client *http.Client) *HttpWrapper {
	return &HttpWrapper{http: client}
}
