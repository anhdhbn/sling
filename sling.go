package sling

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"

	goquery "github.com/google/go-querystring/query"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	jsonContentType = "application/json"
	formContentType = "application/x-www-form-urlencoded"
)

const (
	MethodGet     = "GET"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodPatch   = "PATCH"
	MethodHead    = "HEAD"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
	MethodConnect = "CONNECT"

	hdrContentTypeKey   = "Content-Type"
	hdrAuthorizationKey = "Authorization"
)

// Doer executes http requests.  It is implemented by *http.Client.  You can
// wrap *http.Client with layers of Doers to form a stack of client-side
// middleware.
type Doer interface {
	Do(req *http.Request) (*http.Response, []byte, error)
}

// Sling is an HTTP Request builder and sender.
type Sling struct {
	// http Client for doing requests
	httpClient Doer
	// HTTP method (GET, POST, etc.)
	method string
	// raw url string for requests
	rawURL string
	// stores key-values pairs to add to request's Headers
	header http.Header
	// url tagged query structs
	queryStructs []interface{}
	queryParams  map[string]string
	// body provider
	bodyProvider BodyProvider
	// response decoder
	responseDecoder ResponseDecoder

	ctx       context.Context
	isSuccess SuccessDecider
}

var defaultClient = NewHttpWrapper(&http.Client{
	Transport: otelhttp.NewTransport(http.DefaultTransport),
})

// New returns a new Sling with an http DefaultClient.
func New() *Sling {
	return &Sling{
		httpClient:      defaultClient,
		method:          MethodGet,
		header:          make(http.Header),
		queryStructs:    make([]interface{}, 0),
		queryParams:     make(map[string]string),
		responseDecoder: jsonDecoder{},
		isSuccess:       DecodeOnSuccess,
	}
}

// New returns a copy of a Sling for creating a new Sling with properties
// from a parent Sling. For example,
//
//	parentSling := sling.New().Client(client).Base("https://api.io/")
//	fooSling := parentSling.New().Get("foo/")
//	barSling := parentSling.New().Get("bar/")
//
// fooSling and barSling will both use the same client, but send requests to
// https://api.io/foo/ and https://api.io/bar/ respectively.
//
// Note that query and body values are copied so if pointer values are used,
// mutating the original value will mutate the value within the child Sling.
func (s *Sling) New() *Sling {
	// copy Headers pairs into new Header map
	headerCopy := make(http.Header)
	for k, v := range s.header {
		headerCopy[k] = v
	}
	return &Sling{
		httpClient:      s.httpClient,
		method:          s.method,
		rawURL:          s.rawURL,
		header:          headerCopy,
		queryStructs:    append([]interface{}{}, s.queryStructs...),
		bodyProvider:    s.bodyProvider,
		queryParams:     s.queryParams,
		responseDecoder: s.responseDecoder,
		isSuccess:       s.isSuccess,
	}
}

// Http Client

// Client sets the http Client used to do requests. If a nil client is given,
// the http.DefaultClient will be used.
func (s *Sling) Client(httpWrapper *HttpWrapper) *Sling {
	if httpWrapper == nil {
		return s.Doer(defaultClient)
	}
	return s.Doer(httpWrapper)
}

// Doer sets the custom Doer implementation used to do requests.
// If a nil client is given, the http.DefaultClient will be used.
func (s *Sling) Doer(doer Doer) *Sling {
	if doer == nil {
		s.httpClient = defaultClient
	} else {
		s.httpClient = doer
	}
	return s
}

// Context method returns the Context if its already set in request
// otherwise it creates new one using `context.Background()`.
func (s *Sling) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *Sling) AutoRetry(opts ...RetryOption) *Sling {
	s.httpClient = NewRetryDoer(s.httpClient, opts...)
	return s
}

// SetContext method sets the context.Context for current Request. It allows
// to interrupt the request execution if ctx.Done() channel is closed.
// See https://blog.golang.org/context article and the "context" package
// documentation.
func (s *Sling) SetContext(ctx context.Context) *Sling {
	s.ctx = ctx
	return s
}

// Method

// Head sets the Sling method to HEAD and sets the given pathURL.
func (s *Sling) Head(pathURL string) *Sling {
	s.method = MethodHead
	return s.Path(pathURL)
}

// Get sets the Sling method to GET and sets the given pathURL.
func (s *Sling) Get(pathURL string) *Sling {
	s.method = MethodGet
	return s.Path(pathURL)
}

// Post sets the Sling method to POST and sets the given pathURL.
func (s *Sling) Post(pathURL string) *Sling {
	s.method = MethodPost
	return s.Path(pathURL)
}

// Put sets the Sling method to PUT and sets the given pathURL.
func (s *Sling) Put(pathURL string) *Sling {
	s.method = MethodPut
	return s.Path(pathURL)
}

// Patch sets the Sling method to PATCH and sets the given pathURL.
func (s *Sling) Patch(pathURL string) *Sling {
	s.method = MethodPatch
	return s.Path(pathURL)
}

// Delete sets the Sling method to DELETE and sets the given pathURL.
func (s *Sling) Delete(pathURL string) *Sling {
	s.method = MethodDelete
	return s.Path(pathURL)
}

// Options sets the Sling method to OPTIONS and sets the given pathURL.
func (s *Sling) Options(pathURL string) *Sling {
	s.method = MethodOptions
	return s.Path(pathURL)
}

// Trace sets the Sling method to TRACE and sets the given pathURL.
func (s *Sling) Trace(pathURL string) *Sling {
	s.method = MethodTrace
	return s.Path(pathURL)
}

// Connect sets the Sling method to CONNECT and sets the given pathURL.
func (s *Sling) Connect(pathURL string) *Sling {
	s.method = MethodConnect
	return s.Path(pathURL)
}

// Header

// Add adds the key, value pair in Headers, appending values for existing keys
// to the key's values. Header keys are canonicalized.
func (s *Sling) AddHeader(key, value string) *Sling {
	s.header.Add(key, value)
	return s
}

// Set sets the key, value pair in Headers, replacing existing values
// associated with key. Header keys are canonicalized.
func (s *Sling) SetHeader(key, value string) *Sling {
	s.header.Set(key, value)
	return s
}

// SetBasicAuth sets the Authorization header to use HTTP Basic Authentication
// with the provided username and password. With HTTP Basic Authentication
// the provided username and password are not encrypted.
func (s *Sling) SetBasicAuth(username, password string) *Sling {
	return s.SetHeader(hdrAuthorizationKey, "Basic "+basicAuth(username, password))
}

// basicAuth returns the base64 encoded username:password for basic auth copied
// from net/http.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// SetBearerAuth sets the Authorization header to use HTTP Bearer Authentication
// with the provided token.
func (s *Sling) SetBearerAuth(token string) *Sling {
	return s.SetHeader(hdrAuthorizationKey, "Bearer "+token)
}

func (s *Sling) WithSuccessDecider(isSuccess SuccessDecider) *Sling {
	s.isSuccess = isSuccess
	return s
}

// Url

// Base sets the rawURL. If you intend to extend the url with Path,
// baseUrl should be specified with a trailing slash.
func (s *Sling) Base(rawURL string) *Sling {
	s.rawURL = rawURL
	return s
}

// Path extends the rawURL with the given path by resolving the reference to
// an absolute URL. If parsing errors occur, the rawURL is left unmodified.
func (s *Sling) Path(path string) *Sling {
	baseURL, baseErr := url.Parse(s.rawURL)
	pathURL, pathErr := url.Parse(path)
	if baseErr == nil && pathErr == nil {
		s.rawURL = baseURL.ResolveReference(pathURL).String()
		if strings.HasSuffix(path, "/") && !strings.HasSuffix(s.rawURL, "/") {
			s.rawURL += "/"
		}
		return s
	}
	return s
}

// QueryStruct appends the queryStruct to the Sling's queryStructs. The value
// pointed to by each queryStruct will be encoded as url query parameters on
// new requests (see Request()).
// The queryStruct argument should be a pointer to a url tagged struct. See
// https://godoc.org/github.com/google/go-querystring/query for details.
func (s *Sling) QueryStruct(queryStruct interface{}) *Sling {
	if queryStruct != nil {
		s.queryStructs = append(s.queryStructs, queryStruct)
	}
	return s
}

func (s *Sling) QueryParams(params map[string]string) *Sling {
	if params != nil {
		s.queryParams = params
	}
	return s
}

// Body

// Body sets the Sling's body. The body value will be set as the Body on new
// requests (see Request()).
// If the provided body is also an io.Closer, the request Body will be closed
// by http.Client methods.
func (s *Sling) Body(body io.Reader) *Sling {
	if body == nil {
		return s
	}
	return s.BodyProvider(bodyProvider{body: body})
}

// BodyProvider sets the Sling's body provider.
func (s *Sling) BodyProvider(body BodyProvider) *Sling {
	if body == nil {
		return s
	}
	s.bodyProvider = body

	ct := body.ContentType()
	if ct != "" {
		s.SetHeader(hdrContentTypeKey, ct)
	}

	return s
}

// BodyJSON sets the Sling's bodyJSON. The value pointed to by the bodyJSON
// will be JSON encoded as the Body on new requests (see Request()).
// The bodyJSON argument should be a pointer to a JSON tagged struct. See
// https://golang.org/pkg/encoding/json/#MarshalIndent for details.
func (s *Sling) BodyJSON(bodyJSON interface{}) *Sling {
	if bodyJSON == nil {
		return s
	}
	return s.BodyProvider(jsonBodyProvider{payload: bodyJSON})
}

// BodyForm sets the Sling's bodyForm. The value pointed to by the bodyForm
// will be url encoded as the Body on new requests (see Request()).
// The bodyForm argument should be a pointer to a url tagged struct. See
// https://godoc.org/github.com/google/go-querystring/query for details.
func (s *Sling) BodyForm(bodyForm interface{}) *Sling {
	if bodyForm == nil {
		return s
	}
	return s.BodyProvider(formBodyProvider{payload: bodyForm})
}

// Requests

// Request returns a new http.Request created with the Sling properties.
// Returns any errors parsing the rawURL, encoding query structs, encoding
// the body, or creating the http.Request.
func (s *Sling) Request() (*http.Request, error) {
	reqURL, err := url.Parse(s.rawURL)
	if err != nil {
		return nil, err
	}

	err = buildQueryParamUrl(reqURL, s.queryStructs, s.queryParams)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if s.bodyProvider != nil {
		body, err = s.bodyProvider.Body()
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(s.Context(), s.method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}
	addHeaders(req, s.header)
	return req, err
}

// buildQueryParamUrl parses url tagged query structs using go-querystring to
// encode them to url.Values and format them onto the url.RawQuery. Any
// query parsing or encoding errors are returned.
func buildQueryParamUrl(reqURL *url.URL, queryStructs []interface{}, queryParams map[string]string) error {
	urlValues, err := url.ParseQuery(reqURL.RawQuery)
	if err != nil {
		return err
	}
	// encodes query structs into a url.Values map and merges maps
	for _, queryStruct := range queryStructs {
		queryValues, err := goquery.Values(queryStruct)
		if err != nil {
			return err
		}
		for key, values := range queryValues {
			for _, value := range values {
				urlValues.Add(key, value)
			}
		}
	}
	for k, v := range queryParams {
		urlValues.Add(k, v)
	}
	// url.Values format to a sorted "url encoded" string, e.g. "key=val&foo=bar"
	reqURL.RawQuery = urlValues.Encode()
	return nil
}

// addHeaders adds the key, value pairs from the given http.Header to the
// request. Values for existing keys are appended to the keys values.
func addHeaders(req *http.Request, header http.Header) {
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

// Sending

// ResponseDecoder sets the Sling's response decoder.
func (s *Sling) ResponseDecoder(decoder ResponseDecoder) *Sling {
	if decoder == nil {
		return s
	}
	s.responseDecoder = decoder
	return s
}

// ReceiveSuccess creates a new HTTP request and returns the response. Success
// responses (2XX) are JSON decoded into the value pointed to by successV.
// Any error creating the request, sending it, or decoding a 2XX response
// is returned.
func (s *Sling) ReceiveSuccess(successV interface{}) (*Response, error) {
	return s.Receive(successV, nil)
}

// Receive creates a new HTTP request and returns the response. Success
// responses (2XX) are JSON decoded into the value pointed to by successV and
// other responses are JSON decoded into the value pointed to by failureV.
// If the status code of response is 204(no content) or the Content-Lenght is 0,
// decoding is skipped. Any error creating the request, sending it, or decoding
// the response is returned.
// Receive is shorthand for calling Request and Do.
func (s *Sling) Receive(successV, failureV interface{}) (*Response, error) {
	req, err := s.Request()
	if err != nil {
		return nil, err
	}
	return s.Do(req, successV, failureV)
}

// Do sends an HTTP request and returns the response. Success responses (2XX)
// are JSON decoded into the value pointed to by successV and other responses
// are JSON decoded into the value pointed to by failureV.
// If the status code of response is 204(no content) or the Content-Length is 0,
// decoding is skipped. Any error sending the request or decoding the response
// is returned.
func (s *Sling) Do(req *http.Request, successV, failureV interface{}) (*Response, error) {
	resp, rawData, err := s.httpClient.Do(req)
	if err != nil {
		return NewResponse(resp, rawData), err
	}

	// Don't try to decode on 204s or Content-Length is 0
	if resp.StatusCode == http.StatusNoContent || resp.ContentLength == 0 {
		return NewResponse(resp, rawData), nil
	}

	// Decode from json
	if successV != nil || failureV != nil {
		err = decodeResponse(resp, rawData, s.isSuccess, s.responseDecoder, successV, failureV)
	}
	return NewResponse(resp, rawData), err
}

// decodeResponse decodes response Body into the value pointed to by successV
// if the response is a success (2XX) or into the value pointed to by failureV
// otherwise. If the successV or failureV argument to decode into is nil,
// decoding is skipped.
// Caller is responsible for closing the resp.Body.
func decodeResponse(resp *http.Response, rawData []byte, isSuccess SuccessDecider, decoder ResponseDecoder, successV, failureV interface{}) error {
	if isSuccess(resp) {
		switch sv := successV.(type) {
		case nil:
			return nil
		case *Raw:
			*sv = rawData
			return nil
		default:
			return decoder.Decode(rawData, successV)
		}
	} else {
		switch fv := failureV.(type) {
		case nil:
			return nil
		case *Raw:
			*fv = rawData
			return nil
		default:
			return decoder.Decode(rawData, failureV)
		}
	}
}
