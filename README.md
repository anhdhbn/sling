# Once upon a time

Modified from: https://github.com/dghubble/sling

Each time, you want to conduct a HTTP request, you must implement like 
```go

func postHTTPSample(ctx context.Context, payload request) (*response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	requestBody := bytes.NewBuffer(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8080/hello", requestBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed")
	}
	var respStruct response
	err = json.NewDecoder(resp.Body).Decode(&respStruct)
	if err != nil {
		return nil, err
	}
	return &respStruct, nil
}
``` 

There are some downsides that we can quickly point out from this code.
- We have to write too much boiler plate verbose code from time to time.
    - That make refactoring changes like adding an tracing event will make a enormous effect to whole codebase.
    - Too much verbose things may leads to mini crackles on details from copy - pasting
    - Harder to maintain such a big code
- Many library need to combine to achieve simple things: "bytes", "context", "fmt", "net/http", "encoding/json".
New developer can't get a fast acquainting curve.  


# Take a nap and no more painful experience

With sling, we can quickly implement the same feature with significantly less verbosity than ever before

```go 
func postSlingSample(ctx context.Context, payload request) (*response, error) {
	var respStruct response

	_, err := sling.New().Base("http://localhost:8080").
		Get("/hello").
		SetContext(ctx).
		BodyJSON(payload).
		WithSuccessDecider(func(h *http.Response) bool {
			return h.StatusCode == http.StatusOK
		}).
		ReceiveSuccess(&respStruct)
	return &respStruct, err
}
``` 

- What will you receive back?
    - Write less code, do more work, be more productive
    - An optimized HTTP dealing experience with best practice included
    - OpenTelemetry tracing & metrics default integrated
    - Extra valuable extension: auto retry, authorization...   
    
# What sling provided

## Initialize
| Function           | Feature                                                                                                                                  |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| New                | Create new sling client                                                                                                                    |
| Doer               | Set a new Doer (replacing http lib client default client with Doer, an interface provide `Do` function)                                  |

## Request builder
### Context builder 
| Function           | Feature                                                                                                                                  |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| Context            | Get the current request context                                                                                                          |
| SetContext         | Do the request with current context                                                                                                      |
| AddHeader          | Add value to current header key                                                                                                          |
| SetHeader          | Replace value for current header key                                                                                                     |
| SetHeaders         | Replace current headers                                                                                                                  |
| SetBasicAuth       | Set up the Basic authorization header                                                                                                    |
| SetAuthToken       | Set up standard Teko Bearer token                                                                                                        |

### Path builder 
| Function           | Feature                                                                                                                                  |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| Base               | Set up base host (use for all request use the same client instance)                                                                      |
| Path               | Extend the URL by the given path                                                                                                         |
| QueryStruct        | Extend the URL by the provided query parameter                                                                                           |

### Body builder
| Function           | Feature                                                                                                                                  |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| Body               | Provide request raw body                                                                                                                 |
| BodyProvider       | Provide request raw body with custom content type                                                                                        |
| BodyJSON           | Provide request body as content type "application/json"                                                                                  |
| BodyForm           | Provide request body as content type "application/x-www-form-urlencoded"                                                                 |

### Response config

| Function           | Feature                                                                                                                                  |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| ResponseDecoder    | Setup response decoder (JSON, XML, raw, etc...)                                                                                          |
| WithSuccessDecider | Change the condition that differentiate if the request is success or not                                                                 |

## Execution
| Function           | Feature                                                                                                                                  |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| Request            | Build request based on provided data                                                                                                     |
| ReceiveSuccess     | Receive and parse the response body using the provided response decoder only if the request is success                                   |
| Receive            | Receive and parse the response body using the provided response decoder if the request is success or failed                              |
| Do                 | Do with custom HTTP request, receive and parse the response body using the provided response decoder if the request is success or failed |

## Extensions

### AutoRetry

#### Default strategy
- min retry wait: 1 sec 
- max retry wait: 30 sec 
- max retry times: 4
- retry policy: 
    - not retry if one of these errors occur: 
        - too much redirects
        - invalid scheme
        - TLS certs invalid. 
    - Otherwise, retry if 
        - status code 429 (server is busy)
        - status code invalid
        - status code over 500 (may related to server outage)
- backoff algorithm: [Exponential Backoff](https://en.wikipedia.org/wiki/Exponential_backoff) with min and max wait time range                                                     

#### Available options

| Function           | Feature                                                                                                                                                                                                                                                                                   |
|--------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------| 
| WithRetryTimes     | Set up the maximum retry times                                                                                                                                                                                                                                                            |
| WithRetryMaxWait   | Set up the maximum wait time before retry the request again                                                                                                                                                                                                                               |
| WithRetryMinWait   | Set up the minimum wait time before retry the request again                                                                                                                                                                                                                               |
| WithRetryPolicy    | Provide alternative retry policy  |
| WithBackoff        | Provide alternative backoff calculate algorithm, Jitter backoff is available for swapping |


# FAQ

1. The success response have reliable struct, but the failed ones are not following any rules. We should handle it case by case. How do sling supports it?
    
    sling supports a raw body response struct for dealing with this problem. Example:
    
    ```go
    var rawBody sling.Raw
    _, err := sling.New().AutoRetry().Base("http://localhost:8080").Get("/hello").ReceiveSuccess(&rawBody) 
    ```
    
    `rawBody` is a go byte slice wrapped all the response body. We can parse, log, do whatever we want with it now.

2. The JSON parser work not as expected. Some fields cannot be parsed.

    We use the standard go `encoding/json` library for encoding and decoding JSON payload. 
    Please recheck if the payload is strictly conform the JSON - go struct type conversion.
    For loosely type conversion, you can do some trick with other external lib like `mitchellh/mapstructure`, 
    implement it as a custom decoder and register it. We should be good to go now.


