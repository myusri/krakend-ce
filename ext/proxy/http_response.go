package proxy

import (
	"compress/gzip"
	"context"
	"github.com/devopsfaith/krakend/proxy"
	"io"
	"net/http"
)

// KeepStatusCodeHTTPResponseParserFactory returns an HTTPResponseParser
// that keeps the status code intact from the backend response.
func KeepStatusCodeHTTPResponseParserFactory(
	cfg proxy.HTTPResponseParserConfig) proxy.HTTPResponseParser {
	return func(ctx context.Context, resp *http.Response) (*proxy.Response, error) {
		defer resp.Body.Close()

		var reader io.ReadCloser
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			reader, _ = gzip.NewReader(resp.Body)
			defer reader.Close()
		default:
			reader = resp.Body
		}

		var data map[string]interface{}
		if err := cfg.Decoder(reader, &data); err != nil {
			return nil, err
		}
		// Relay the status code from the HTTP response to the backend
		// response.
		newResponse := proxy.Response{
			Data:       data,
			IsComplete: true,
			Metadata:   proxy.Metadata{StatusCode: resp.StatusCode},
		}
		newResponse = cfg.EntityFormatter.Format(newResponse)
		return &newResponse, nil
	}
}
