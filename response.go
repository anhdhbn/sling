package sling

import (
	"net/http"
)

// Raw is response's raw data
type Raw []byte

// Response is a http response wrapper
type Response struct {
	*http.Response
}

func NewResponse(response *http.Response) *Response {
	return &Response{
		Response: response,
	}
}

// SuccessDecider decide should we decode the response or not
type SuccessDecider func(*http.Response) bool

// DecodeOnSuccess decide that we should decode on success response (http code 2xx)
func DecodeOnSuccess(resp *http.Response) bool {
	return 200 <= resp.StatusCode && resp.StatusCode <= 299
}
