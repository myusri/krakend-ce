package krakend

import (
	botdetector "github.com/devopsfaith/krakend-botdetector/gin"
	extRouter "github.com/devopsfaith/krakend-ce/ext/router/gin"
	"github.com/devopsfaith/krakend-jose"
	ginjose "github.com/devopsfaith/krakend-jose/gin"
	lua "github.com/devopsfaith/krakend-lua/router/gin"
	metrics "github.com/devopsfaith/krakend-metrics/gin"
	opencensus "github.com/devopsfaith/krakend-opencensus/router/gin"
	juju "github.com/devopsfaith/krakend-ratelimit/juju/router/gin"
	"github.com/devopsfaith/krakend/logging"
	router "github.com/devopsfaith/krakend/router/gin"
)

// NewHandlerFactory make use of our own base endpoint handler
// extRouter.EndpointHandler which would return the status code from the
// backends to the endpoint caller.
func NewHandlerFactory(logger logging.Logger, metricCollector *metrics.Metrics, rejecter jose.RejecterFactory) router.HandlerFactory {
	handlerFactory := extRouter.EndpointHandler
	handlerFactory = juju.NewRateLimiterMw(handlerFactory)
	handlerFactory = lua.HandlerFactory(logger, handlerFactory)
	handlerFactory = ginjose.HandlerFactory(handlerFactory, logger, rejecter)
	handlerFactory = metricCollector.NewHTTPHandlerFactory(handlerFactory)
	handlerFactory = opencensus.New(handlerFactory)
	handlerFactory = botdetector.New(handlerFactory, logger)
	return handlerFactory
}

type handlerFactory struct{}

func (h handlerFactory) NewHandlerFactory(l logging.Logger, m *metrics.Metrics, r jose.RejecterFactory) router.HandlerFactory {
	return NewHandlerFactory(l, m, r)
}
