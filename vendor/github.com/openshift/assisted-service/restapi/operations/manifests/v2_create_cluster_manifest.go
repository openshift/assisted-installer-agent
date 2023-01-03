// Code generated by go-swagger; DO NOT EDIT.

package manifests

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// V2CreateClusterManifestHandlerFunc turns a function with the right signature into a v2 create cluster manifest handler
type V2CreateClusterManifestHandlerFunc func(V2CreateClusterManifestParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn V2CreateClusterManifestHandlerFunc) Handle(params V2CreateClusterManifestParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// V2CreateClusterManifestHandler interface for that can handle valid v2 create cluster manifest params
type V2CreateClusterManifestHandler interface {
	Handle(V2CreateClusterManifestParams, interface{}) middleware.Responder
}

// NewV2CreateClusterManifest creates a new http.Handler for the v2 create cluster manifest operation
func NewV2CreateClusterManifest(ctx *middleware.Context, handler V2CreateClusterManifestHandler) *V2CreateClusterManifest {
	return &V2CreateClusterManifest{Context: ctx, Handler: handler}
}

/*
	V2CreateClusterManifest swagger:route POST /v2/clusters/{cluster_id}/manifests manifests v2CreateClusterManifest

Creates a manifest for customizing cluster installation.
*/
type V2CreateClusterManifest struct {
	Context *middleware.Context
	Handler V2CreateClusterManifestHandler
}

func (o *V2CreateClusterManifest) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewV2CreateClusterManifestParams()
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
