// Code generated by go-swagger; DO NOT EDIT.

package installer

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// V2ListFeatureSupportLevelsHandlerFunc turns a function with the right signature into a v2 list feature support levels handler
type V2ListFeatureSupportLevelsHandlerFunc func(V2ListFeatureSupportLevelsParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn V2ListFeatureSupportLevelsHandlerFunc) Handle(params V2ListFeatureSupportLevelsParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// V2ListFeatureSupportLevelsHandler interface for that can handle valid v2 list feature support levels params
type V2ListFeatureSupportLevelsHandler interface {
	Handle(V2ListFeatureSupportLevelsParams, interface{}) middleware.Responder
}

// NewV2ListFeatureSupportLevels creates a new http.Handler for the v2 list feature support levels operation
func NewV2ListFeatureSupportLevels(ctx *middleware.Context, handler V2ListFeatureSupportLevelsHandler) *V2ListFeatureSupportLevels {
	return &V2ListFeatureSupportLevels{Context: ctx, Handler: handler}
}

/*
	V2ListFeatureSupportLevels swagger:route GET /v2/feature-support-levels installer v2ListFeatureSupportLevels

(DEPRECATED) Retrieves the support levels for features for each OpenShift version.
*/
type V2ListFeatureSupportLevels struct {
	Context *middleware.Context
	Handler V2ListFeatureSupportLevelsHandler
}

func (o *V2ListFeatureSupportLevels) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewV2ListFeatureSupportLevelsParams()
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
