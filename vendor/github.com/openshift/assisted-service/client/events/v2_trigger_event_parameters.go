// Code generated by go-swagger; DO NOT EDIT.

package events

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/openshift/assisted-service/models"
)

// NewV2TriggerEventParams creates a new V2TriggerEventParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewV2TriggerEventParams() *V2TriggerEventParams {
	return &V2TriggerEventParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewV2TriggerEventParamsWithTimeout creates a new V2TriggerEventParams object
// with the ability to set a timeout on a request.
func NewV2TriggerEventParamsWithTimeout(timeout time.Duration) *V2TriggerEventParams {
	return &V2TriggerEventParams{
		timeout: timeout,
	}
}

// NewV2TriggerEventParamsWithContext creates a new V2TriggerEventParams object
// with the ability to set a context for a request.
func NewV2TriggerEventParamsWithContext(ctx context.Context) *V2TriggerEventParams {
	return &V2TriggerEventParams{
		Context: ctx,
	}
}

// NewV2TriggerEventParamsWithHTTPClient creates a new V2TriggerEventParams object
// with the ability to set a custom HTTPClient for a request.
func NewV2TriggerEventParamsWithHTTPClient(client *http.Client) *V2TriggerEventParams {
	return &V2TriggerEventParams{
		HTTPClient: client,
	}
}

/*
V2TriggerEventParams contains all the parameters to send to the API endpoint

	for the v2 trigger event operation.

	Typically these are written to a http.Request.
*/
type V2TriggerEventParams struct {

	/* TriggerEventParams.

	   The event to be created.
	*/
	TriggerEventParams *models.Event

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the v2 trigger event params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *V2TriggerEventParams) WithDefaults() *V2TriggerEventParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the v2 trigger event params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *V2TriggerEventParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the v2 trigger event params
func (o *V2TriggerEventParams) WithTimeout(timeout time.Duration) *V2TriggerEventParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the v2 trigger event params
func (o *V2TriggerEventParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the v2 trigger event params
func (o *V2TriggerEventParams) WithContext(ctx context.Context) *V2TriggerEventParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the v2 trigger event params
func (o *V2TriggerEventParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the v2 trigger event params
func (o *V2TriggerEventParams) WithHTTPClient(client *http.Client) *V2TriggerEventParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the v2 trigger event params
func (o *V2TriggerEventParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithTriggerEventParams adds the triggerEventParams to the v2 trigger event params
func (o *V2TriggerEventParams) WithTriggerEventParams(triggerEventParams *models.Event) *V2TriggerEventParams {
	o.SetTriggerEventParams(triggerEventParams)
	return o
}

// SetTriggerEventParams adds the triggerEventParams to the v2 trigger event params
func (o *V2TriggerEventParams) SetTriggerEventParams(triggerEventParams *models.Event) {
	o.TriggerEventParams = triggerEventParams
}

// WriteToRequest writes these params to a swagger request
func (o *V2TriggerEventParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.TriggerEventParams != nil {
		if err := r.SetBodyParam(o.TriggerEventParams); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
