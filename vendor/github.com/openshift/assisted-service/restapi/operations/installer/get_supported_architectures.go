// Code generated by go-swagger; DO NOT EDIT.

package installer

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/openshift/assisted-service/models"
)

// GetSupportedArchitecturesHandlerFunc turns a function with the right signature into a get supported architectures handler
type GetSupportedArchitecturesHandlerFunc func(GetSupportedArchitecturesParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn GetSupportedArchitecturesHandlerFunc) Handle(params GetSupportedArchitecturesParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// GetSupportedArchitecturesHandler interface for that can handle valid get supported architectures params
type GetSupportedArchitecturesHandler interface {
	Handle(GetSupportedArchitecturesParams, interface{}) middleware.Responder
}

// NewGetSupportedArchitectures creates a new http.Handler for the get supported architectures operation
func NewGetSupportedArchitectures(ctx *middleware.Context, handler GetSupportedArchitecturesHandler) *GetSupportedArchitectures {
	return &GetSupportedArchitectures{Context: ctx, Handler: handler}
}

/*
	GetSupportedArchitectures swagger:route GET /v2/support-levels/architectures installer getSupportedArchitectures

Retrieves the architecture support-levels for each OpenShift version.
*/
type GetSupportedArchitectures struct {
	Context *middleware.Context
	Handler GetSupportedArchitecturesHandler
}

func (o *GetSupportedArchitectures) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetSupportedArchitecturesParams()
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

// GetSupportedArchitecturesOKBody get supported architectures o k body
//
// swagger:model GetSupportedArchitecturesOKBody
type GetSupportedArchitecturesOKBody struct {

	// Keys will be one of architecture-support-level-id enum.
	Architectures models.SupportLevels `json:"architectures,omitempty"`
}

// Validate validates this get supported architectures o k body
func (o *GetSupportedArchitecturesOKBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateArchitectures(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetSupportedArchitecturesOKBody) validateArchitectures(formats strfmt.Registry) error {
	if swag.IsZero(o.Architectures) { // not required
		return nil
	}

	if o.Architectures != nil {
		if err := o.Architectures.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("getSupportedArchitecturesOK" + "." + "architectures")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("getSupportedArchitecturesOK" + "." + "architectures")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this get supported architectures o k body based on the context it is used
func (o *GetSupportedArchitecturesOKBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := o.contextValidateArchitectures(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetSupportedArchitecturesOKBody) contextValidateArchitectures(ctx context.Context, formats strfmt.Registry) error {

	if err := o.Architectures.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("getSupportedArchitecturesOK" + "." + "architectures")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("getSupportedArchitecturesOK" + "." + "architectures")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (o *GetSupportedArchitecturesOKBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *GetSupportedArchitecturesOKBody) UnmarshalBinary(b []byte) error {
	var res GetSupportedArchitecturesOKBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
