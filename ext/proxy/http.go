package proxy

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	extClient "github.com/devopsfaith/krakend-ce/ext/transport/http/client"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/encoding"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/transport/http/client"
)

// NewHTTPProxyWithHTTPExecutor makes use of our own HTTPResponseParser and
// HTTPStatusHandler as return by KeepStatusCodeHTTPResponseParserFactory and
// GetHTTPStatusHandler, respectively.
func NewHTTPProxyWithHTTPExecutor(
	remote *config.Backend, re client.HTTPRequestExecutor,
	dec encoding.Decoder) proxy.Proxy {
	if remote.Encoding == encoding.NOOP {
		return NewHTTPProxyKeepStatusCode(
			remote, re, client.NoOpHTTPStatusHandler,
			proxy.NoOpHTTPResponseParser)
	}

	ef := proxy.NewEntityFormatter(remote)
	rp := KeepStatusCodeHTTPResponseParserFactory(
		proxy.HTTPResponseParserConfig{Decoder: dec, EntityFormatter: ef})
	return NewHTTPProxyKeepStatusCode(
		remote, re, extClient.GetHTTPStatusHandler(remote), rp)
}

// NewHTTPProxyKeepStatusCode is our replacement to proxy.NewHTTPProxyDetailed
// because we keep and return status code from the backend proxy.
func NewHTTPProxyKeepStatusCode(
	_ *config.Backend, re client.HTTPRequestExecutor,
	ch client.HTTPStatusHandler, rp proxy.HTTPResponseParser) proxy.Proxy {

	// There is a bit of a sticky situation here. The incoming request is from
	// the endpoint and we are technically a server serving the endpoint. We
	// therefore should not be closing the endpoint request body as we pass it
	// to the backend. So, we should wrap the incoming request in
	// ioutil.NoCloser?
	// Also, since GET requests is not supposed to out a request body (TM), we
	//will not pass the endpoint body to GET requests. There is that problem of
	// multiple backends with multiple POST requests. You are on your own on
	// that one.
	return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
		body := request.Body // ioutil.NopCloser(request.Body)
		method := strings.ToTitle(request.Method)
		if method == "GET" || method == "HEAD" {
			body = nil
		}
		requestToBakend, err := http.NewRequest(method, request.URL.String(), body)
		if err != nil {
			return nil, err
		}
		requestToBakend.Header = make(map[string][]string, len(request.Headers))
		for k, vs := range request.Headers {
			tmp := make([]string, len(vs))
			copy(tmp, vs)
			requestToBakend.Header[k] = tmp
		}
		if request.Body != nil {
			if v, ok := request.Headers["Content-Length"]; ok && len(v) == 1 && v[0] != "chunked" {
				if size, err := strconv.Atoi(v[0]); err == nil {
					requestToBakend.ContentLength = int64(size)
				}
			}
		}

		resp, err := re(ctx, requestToBakend)
		if requestToBakend.Body != nil {
			requestToBakend.Body.Close()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if err != nil {
			return nil, err
		}

		resp, err = ch(ctx, resp)
		if err != nil {
			// Unexpected system error. For our case, we convert unexpected
			// Error into a response with status code 500.
			return &proxy.Response{
				Data: map[string]interface{}{
					"error": err.Error(),
				},
				Metadata: proxy.Metadata{
					StatusCode: http.StatusInternalServerError,
				},
			}, nil
		}

		return rp(ctx, resp)
	}
}
