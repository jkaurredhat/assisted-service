// Code generated by go-swagger; DO NOT EDIT.

package installer

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime/middleware"
)

// NewV2ListFeatureSupportLevelsParams creates a new V2ListFeatureSupportLevelsParams object
// no default values defined in spec.
func NewV2ListFeatureSupportLevelsParams() V2ListFeatureSupportLevelsParams {

	return V2ListFeatureSupportLevelsParams{}
}

// V2ListFeatureSupportLevelsParams contains all the bound params for the v2 list feature support levels operation
// typically these are obtained from a http.Request
//
// swagger:parameters v2ListFeatureSupportLevels
type V2ListFeatureSupportLevelsParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewV2ListFeatureSupportLevelsParams() beforehand.
func (o *V2ListFeatureSupportLevelsParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}