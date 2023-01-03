// Code generated by go-swagger; DO NOT EDIT.

package installer

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
)

// NewV2GetPresignedForClusterCredentialsParams creates a new V2GetPresignedForClusterCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewV2GetPresignedForClusterCredentialsParams() *V2GetPresignedForClusterCredentialsParams {
	return &V2GetPresignedForClusterCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewV2GetPresignedForClusterCredentialsParamsWithTimeout creates a new V2GetPresignedForClusterCredentialsParams object
// with the ability to set a timeout on a request.
func NewV2GetPresignedForClusterCredentialsParamsWithTimeout(timeout time.Duration) *V2GetPresignedForClusterCredentialsParams {
	return &V2GetPresignedForClusterCredentialsParams{
		timeout: timeout,
	}
}

// NewV2GetPresignedForClusterCredentialsParamsWithContext creates a new V2GetPresignedForClusterCredentialsParams object
// with the ability to set a context for a request.
func NewV2GetPresignedForClusterCredentialsParamsWithContext(ctx context.Context) *V2GetPresignedForClusterCredentialsParams {
	return &V2GetPresignedForClusterCredentialsParams{
		Context: ctx,
	}
}

// NewV2GetPresignedForClusterCredentialsParamsWithHTTPClient creates a new V2GetPresignedForClusterCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewV2GetPresignedForClusterCredentialsParamsWithHTTPClient(client *http.Client) *V2GetPresignedForClusterCredentialsParams {
	return &V2GetPresignedForClusterCredentialsParams{
		HTTPClient: client,
	}
}

/*
V2GetPresignedForClusterCredentialsParams contains all the parameters to send to the API endpoint

	for the v2 get presigned for cluster credentials operation.

	Typically these are written to a http.Request.
*/
type V2GetPresignedForClusterCredentialsParams struct {

	/* ClusterID.

	   The cluster that owns the file that should be downloaded.

	   Format: uuid
	*/
	ClusterID strfmt.UUID

	/* FileName.

	   The file to be downloaded.
	*/
	FileName string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the v2 get presigned for cluster credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *V2GetPresignedForClusterCredentialsParams) WithDefaults() *V2GetPresignedForClusterCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the v2 get presigned for cluster credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *V2GetPresignedForClusterCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) WithTimeout(timeout time.Duration) *V2GetPresignedForClusterCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) WithContext(ctx context.Context) *V2GetPresignedForClusterCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) WithHTTPClient(client *http.Client) *V2GetPresignedForClusterCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) WithClusterID(clusterID strfmt.UUID) *V2GetPresignedForClusterCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) SetClusterID(clusterID strfmt.UUID) {
	o.ClusterID = clusterID
}

// WithFileName adds the fileName to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) WithFileName(fileName string) *V2GetPresignedForClusterCredentialsParams {
	o.SetFileName(fileName)
	return o
}

// SetFileName adds the fileName to the v2 get presigned for cluster credentials params
func (o *V2GetPresignedForClusterCredentialsParams) SetFileName(fileName string) {
	o.FileName = fileName
}

// WriteToRequest writes these params to a swagger request
func (o *V2GetPresignedForClusterCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID.String()); err != nil {
		return err
	}

	// query param file_name
	qrFileName := o.FileName
	qFileName := qrFileName
	if qFileName != "" {

		if err := r.SetQueryParam("file_name", qFileName); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
