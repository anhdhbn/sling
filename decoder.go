package sling

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ResponseDecoder decodes http responses into struct values.
type ResponseDecoder interface {
	// Decode decodes the response into the value pointed to by v.
	Decode(bytes []byte, v interface{}) error
}

// jsonDecoder decodes http response JSON into a JSON-tagged struct value.
type jsonDecoder struct {
}

// Decode decodes the Response Body into the value pointed to by v.
// Caller must provide a non-nil v and close the resp.Body.
func (d jsonDecoder) Decode(bytes []byte, v interface{}) error {
	return json.Unmarshal(bytes, v)
}

// jsonDecoder decodes http response JSON into a JSON-tagged struct value.
type JsonpbDecoder struct {
}

// Decode decodes the Response Body into the value pointed to by v.
// Caller must provide a non-nil v and close the resp.Body.
func (d JsonpbDecoder) Decode(bytes []byte, v interface{}) error {
	return protojson.Unmarshal(bytes, v.(proto.Message))
}
