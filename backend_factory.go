package krakend

import (
	"context"

	amqp "github.com/devopsfaith/krakend-amqp"
	extProxy "github.com/devopsfaith/krakend-ce/ext/proxy"
	cel "github.com/devopsfaith/krakend-cel"
	cb "github.com/devopsfaith/krakend-circuitbreaker/gobreaker/proxy"
	httpcache "github.com/devopsfaith/krakend-httpcache"
	lambda "github.com/devopsfaith/krakend-lambda"
	lua "github.com/devopsfaith/krakend-lua/proxy"
	"github.com/devopsfaith/krakend-martian"
	metrics "github.com/devopsfaith/krakend-metrics/gin"
	"github.com/devopsfaith/krakend-oauth2-clientcredentials"
	opencensus "github.com/devopsfaith/krakend-opencensus"
	pubsub "github.com/devopsfaith/krakend-pubsub"
	juju "github.com/devopsfaith/krakend-ratelimit/juju/proxy"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/transport/http/client"
	httprequestexecutor "github.com/devopsfaith/krakend/transport/http/client/plugin"
)

// NewBackendFactory creates a BackendFactory by stacking all the available middlewares:
// - oauth2 client credentials
// - http cache
// - martian
// - pubsub
// - amqp
// - cel
// - lua
// - rate-limit
// - circuit breaker
// - metrics collector
// - opencensus collector
func NewBackendFactory(logger logging.Logger, metricCollector *metrics.Metrics) proxy.BackendFactory {
	return NewBackendFactoryWithContext(context.Background(), logger, metricCollector)
}

// NewBackendFactory creates a BackendFactory by stacking all the available middlewares and injecting the received context
func NewBackendFactoryWithContext(ctx context.Context, logger logging.Logger, metricCollector *metrics.Metrics) proxy.BackendFactory {
	requestExecutorFactory := func(cfg *config.Backend) client.HTTPRequestExecutor {
		var clientFactory client.HTTPClientFactory
		if _, ok := cfg.ExtraConfig[oauth2client.Namespace]; ok {
			clientFactory = oauth2client.NewHTTPClient(cfg)
		} else {
			clientFactory = httpcache.NewHTTPClient(cfg)
		}
		return opencensus.HTTPRequestExecutor(clientFactory)
	}
	requestExecutorFactory = httprequestexecutor.HTTPRequestExecutor(logger, requestExecutorFactory)

	// We need to use our own HTTPResponseParser and HTTPStatusHandler because
	// by default an error will be reported if the status code is not 200 or
	// 201. We will still call martian.NewConfiguredBackendFactory for the
	// side-effect (parse.Register)
	martian.NewConfiguredBackendFactory(logger, requestExecutorFactory)
	backendFactory := func(remote *config.Backend) proxy.Proxy {
		re := requestExecutorFactory(remote)
		result, ok := martian.ConfigGetter(remote.ExtraConfig).(martian.Result)
		if !ok {
			return extProxy.NewHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		}
		switch result.Err {
		case nil:
			return extProxy.NewHTTPProxyWithHTTPExecutor(
				remote, martian.HTTPRequestExecutor(result.Result, re), remote.Decoder)
		case martian.ErrEmptyValue:
			return extProxy.NewHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		default:
			logger.Error(result, remote.ExtraConfig)
			return extProxy.NewHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		}
	}

	bf := pubsub.NewBackendFactory(ctx, logger, backendFactory)
	backendFactory = bf.New
	backendFactory = amqp.NewBackendFactory(ctx, logger, backendFactory)
	backendFactory = lambda.BackendFactory(backendFactory)
	backendFactory = cel.BackendFactory(logger, backendFactory)
	backendFactory = lua.BackendFactory(logger, backendFactory)
	backendFactory = juju.BackendFactory(backendFactory)
	backendFactory = cb.BackendFactory(backendFactory, logger)
	backendFactory = metricCollector.BackendFactory("backend", backendFactory)
	backendFactory = opencensus.BackendFactory(backendFactory)
	return backendFactory
}

type backendFactory struct{}

func (b backendFactory) NewBackendFactory(ctx context.Context, l logging.Logger, m *metrics.Metrics) proxy.BackendFactory {
	return NewBackendFactoryWithContext(ctx, l, m)
}
