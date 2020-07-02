package gin

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/router"
	ginRouter "github.com/devopsfaith/krakend/router/gin"
	"github.com/devopsfaith/krakend/transport/http/server"
)

// NewFactory returns a gin router factory with the injected configuration.
// We are replacing the default Gin router NewFactory so that we can register
// the Gin router without enforcing the only one backend for non-GET endpoint.
func NewFactory(cfg ginRouter.Config) router.Factory {
	return factory{cfg}
}

type factory struct {
	cfg ginRouter.Config
}

func (rf factory) New() router.Router {
	return rf.NewWithContext(context.Background())
}

func (rf factory) NewWithContext(ctx context.Context) router.Router {
	return extGinRouter{rf.cfg, ctx, rf.cfg.RunServer}
}

// We would like to use our own Gin router
type extGinRouter struct {
	cfg       ginRouter.Config
	ctx       context.Context
	RunServer ginRouter.RunServerFunc
}

// We pretty much copy the Gin router Run implementation.
func (r extGinRouter) Run(cfg config.ServiceConfig) {
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	} else {
		r.cfg.Logger.Debug("Debug enabled")
	}

	server.InitHTTPDefaultTransport(cfg)

	r.cfg.Engine.RedirectTrailingSlash = true
	r.cfg.Engine.RedirectFixedPath = true
	r.cfg.Engine.HandleMethodNotAllowed = true

	r.cfg.Engine.Use(r.cfg.Middlewares...)

	if cfg.Debug {
		r.cfg.Engine.Any("/__debug/*param", ginRouter.DebugHandler(r.cfg.Logger))
	}

	r.registerKrakendEndpoints(cfg.Endpoints)

	r.cfg.Engine.NoRoute(func(c *gin.Context) {
		c.Header(server.CompleteResponseHeaderName, server.HeaderIncompleteResponseValue)
	})

	if err := r.RunServer(r.ctx, cfg, r.cfg.Engine); err != nil {
		r.cfg.Logger.Error(err.Error())
	}

	r.cfg.Logger.Info("Router execution ended")
}

func (r extGinRouter) registerKrakendEndpoints(endpoints []*config.EndpointConfig) {
	for _, c := range endpoints {
		proxyStack, err := r.cfg.ProxyFactory.New(c)
		if err != nil {
			r.cfg.Logger.Error("calling the ProxyFactory", err.Error())
			continue
		}

		r.registerKrakendEndpoint(c.Method, c.Endpoint, r.cfg.HandlerFactory(c, proxyStack))
	}
}

// In the original Gin router `registerKrakendEndpoint`, a check is made for
// non-GET routes with multiple endpoints.
func (r extGinRouter) registerKrakendEndpoint(method, path string, handler gin.HandlerFunc) {
	method = strings.ToTitle(method)
	switch method {
	case http.MethodGet:
		r.cfg.Engine.GET(path, handler)
	case http.MethodPost:
		r.cfg.Engine.POST(path, handler)
	case http.MethodPut:
		r.cfg.Engine.PUT(path, handler)
	case http.MethodPatch:
		r.cfg.Engine.PATCH(path, handler)
	case http.MethodDelete:
		r.cfg.Engine.DELETE(path, handler)
	default:
		r.cfg.Logger.Error("Unsupported method", method)
	}
}
