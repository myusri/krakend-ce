package proxy

import (
	"fmt"
	"github.com/devopsfaith/krakend/proxy"
	"net/http"
)

func init() {
	RegisterCombiners()
}

// RegisterCombiners registers two additional response combiners,
// "default-debug" and "first-error". The first is similar to the "default"
// combiner except it outputs some debug messages. The second returns the
// first non-success code response as the response if the responses from the
// multiple backends has non-success code responses.
func RegisterCombiners() {
	proxy.RegisterResponseCombiner("default-debug", debugDefault)
	proxy.RegisterResponseCombiner("first-error", firstError)
}

var combiners = proxy.NewRegister()
var defaultCombiner, _ = combiners.GetResponseCombiner("default")

// firstError returns the first error response among the many backend responses
func firstError(total int, parts []*proxy.Response) *proxy.Response {
	for _, r := range parts {
		var sc = r.Metadata.StatusCode
		if sc != 0 && sc != http.StatusOK && sc != http.StatusCreated {
			return r
		}
	}
	return defaultCombiner(total, parts)
}

func debugDefault(total int, parts []*proxy.Response) *proxy.Response {
	for i, r := range parts {
		fmt.Println(">> response", i, "status:", r.Metadata.StatusCode)
	}
	return defaultCombiner(total, parts)
}
