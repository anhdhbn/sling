package sling

import (
	"encoding/json"
	"io"
)

// ResponseDecoder decodes http responses into struct values.
type ResponseDecoder interface {
	// Decode decodes the response into the value pointed to by v.
	Decode(resp *Response, v interface{}) error
}

// jsonDecoder decodes http response JSON into a JSON-tagged struct value.
type JsonDecoder struct {
}

// Decode decodes the Response Body into the value pointed to by v.
// Caller must provide a non-nil v and close the resp.Body.
func (d JsonDecoder) Decode(resp *Response, v interface{}) error {
	return json.NewDecoder(resp.Body).Decode(v)
}

type JsonMarshalDecoder struct {
}

// Decode decodes the Response Body into the value pointed to by v.
// Caller must provide a non-nil v and close the resp.Body.
func (d JsonMarshalDecoder) Decode(resp *Response, v interface{}) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Raw = data
	defer resp.Body.Close()
	return json.Unmarshal(data, v)
}
