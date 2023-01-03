// Code generated by go-swagger; DO NOT EDIT.

package installer

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// V2GetClusterDefaultConfigHandlerFunc turns a function with the right signature into a v2 get cluster default config handler
type V2GetClusterDefaultConfigHandlerFunc func(V2GetClusterDefaultConfigParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn V2GetClusterDefaultConfigHandlerFunc) Handle(params V2GetClusterDefaultConfigParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// V2GetClusterDefaultConfigHandler interface for that can handle valid v2 get cluster default config params
type V2GetClusterDefaultConfigHandler interface {
	Handle(V2GetClusterDefaultConfigParams, interface{}) middleware.Responder
}

// NewV2GetClusterDefaultConfig creates a new http.Handler for the v2 get cluster default config operation
func NewV2GetClusterDefaultConfig(ctx *middleware.Context, handler V2GetClusterDefaultConfigHandler) *V2GetClusterDefaultConfig {
	return &V2GetClusterDefaultConfig{Context: ctx, Handler: handler}
}

/*
	V2GetClusterDefaultConfig swagger:route GET /v2/clusters/default-config installer v2GetClusterDefaultConfig

Get the default values for various cluster properties.
*/
type V2GetClusterDefaultConfig struct {
	Context *middleware.Context
	Handler V2GetClusterDefaultConfigHandler
}

func (o *V2GetClusterDefaultConfig) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewV2GetClusterDefaultConfigParams()
	uprinc, aCtx, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	if aCtx != nil {
		*r = *aCtx
	}
	var principal interface{}
	if uprinc != nil {
		principal = uprinc.(interface{}) // this is really a interface{}, I promise
	}

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params, principal) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
