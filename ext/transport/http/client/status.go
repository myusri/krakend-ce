package client

import (
	"context"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/transport/http/client"
	"net/http"
)

// ExtHTTPStatusHandler is an HTTPStatusHandler to replace the
// DefaultHTTPStatusHandler. We choose not to mess with the response in any
// way. As such, we will also ignore "return_error_details" because
// essentially we are showing the error response unchanged.
//
// The DefaultHTTPStatusHandler in krakend/transport/http/client/status.go
// will always return error ErrInvalidStatusCode if the HTTP status code of
// the response is not 200 or 201 (Status OK or Created). With this setup,
// by default, for multi-backend endpoint, successful responses will be merged
// and returned with status 200. Error responses are totally suppressed from
// being merged with the final response for the endpoint.
//
// If we have the "return_error_details" set, we will get a JSON describing
// the the status code and the error response body (serialized in a string)
// for every error response. This starts in DetailedHTTPStatusHandler by
// returning both the response and an HTTPResponseError. The error is caught
// in NewHTTPProxyDetailed and converted into the final response.
func ExtHTTPStatusHandler(_ context.Context, resp *http.Response) (*http.Response, error) {
	return resp, nil
}

func GetHTTPStatusHandler(_ *config.Backend) client.HTTPStatusHandler {
	return ExtHTTPStatusHandler
}
