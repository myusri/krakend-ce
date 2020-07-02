package proxy

import (
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
		return proxy.NewHTTPProxyDetailed(
			remote, re, client.NoOpHTTPStatusHandler,
			proxy.NoOpHTTPResponseParser)
	}

	ef := proxy.NewEntityFormatter(remote)
	rp := KeepStatusCodeHTTPResponseParserFactory(
		proxy.HTTPResponseParserConfig{Decoder: dec, EntityFormatter: ef})
	return proxy.NewHTTPProxyDetailed(
		remote, re, extClient.GetHTTPStatusHandler(remote), rp)
}
