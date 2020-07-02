package gin

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/core"
	"github.com/devopsfaith/krakend/proxy"
	ginRouter "github.com/devopsfaith/krakend/router/gin"
	"github.com/devopsfaith/krakend/transport/http/server"
	"net/http"
)

// EndpointHandler is called by the NewHandlerFactory which is used to prepare
// the base endpoint handler in a stack of handler functions for an endpoint.
func EndpointHandler(configuration *config.EndpointConfig, proxy proxy.Proxy) gin.HandlerFunc {
	return CustomErrorEndpointHandler(configuration, proxy, server.DefaultToHTTPError)
}

// CustomErrorEndpointHandler return a handler function that follows the
// status code from the backend response. We have a "first-error" response
// combiner whereby the first non-success error code response from one of
// the backends is used as the response for the endpoint.
func CustomErrorEndpointHandler(
	configuration *config.EndpointConfig,
	prxy proxy.Proxy, errF server.ToHTTPError) gin.HandlerFunc {
	cacheControlHeaderValue := fmt.Sprintf(
		"public, max-age=%d", int(configuration.CacheTTL.Seconds()))
	isCacheEnabled := configuration.CacheTTL.Seconds() != 0
	requestGenerator := ginRouter.NewRequest(configuration.HeadersToPass)
	render := jsonRender // getRender(configuration)

	return func(c *gin.Context) {
		requestCtx, cancel := context.WithTimeout(c, configuration.Timeout)

		c.Header(core.KrakendHeaderName, core.KrakendHeaderValue)

		response, err := prxy(requestCtx, requestGenerator(c, configuration.QueryString))

		select {
		case <-requestCtx.Done():
			if err == nil {
				err = server.ErrInternalError
			}
		default:
		}

		complete := server.HeaderIncompleteResponseValue

		if response != nil && len(response.Data) > 0 {
			if response.IsComplete {
				complete = server.HeaderCompleteResponseValue
				if isCacheEnabled {
					c.Header("Cache-Control", cacheControlHeaderValue)
				}
			}

			for k, vs := range response.Metadata.Headers {
				for _, v := range vs {
					c.Writer.Header().Add(k, v)
				}
			}
		}

		c.Header(server.CompleteResponseHeaderName, complete)

		if err != nil {
			c.Error(err)

			if response == nil {
				if t, ok := err.(ResponseError); ok {
					c.Status(t.StatusCode())
				} else {
					c.Status(errF(err))
				}
				cancel()
				return
			}
		}
		// If the response collected from the backend(s) signals a non-success
		// status code, use that status code in for the final response.
		sc := response.Metadata.StatusCode
		if sc != http.StatusOK && sc != http.StatusCreated {
			c.Status(sc)
		}
		render(c, response)
		cancel()
	}
}

type ResponseError interface {
	error
	StatusCode() int
}

// TODO: Sorry. Gin render stuff is too tight to access. We will do JSON render
//       only.
var emptyResponse = gin.H{}

func jsonRender(c *gin.Context, response *proxy.Response) {
	status := c.Writer.Status()
	if response == nil {
		c.JSON(status, emptyResponse)
		return
	}
	c.JSON(status, response.Data)
}
