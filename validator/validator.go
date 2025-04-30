// Package validator exposes validation functionality for the
// resource definition.
package validator

import (
	"fmt"
	"regexp"

	"github.com/aep-dev/aep-lib-go/pkg/api"
	"github.com/aep-dev/aep-lib-go/pkg/openapi"
)

const (
	RESOURCE_KIND_REGEX_STRING = "[a-z]+[a-zA-Z0-9]*"
)

var RESOURCE_KIND_REGEX regexp.Regexp

func init() {
	// compile regex via init rather than const to use the other string
	// constants.
	RESOURCE_KIND_REGEX = *regexp.MustCompile(RESOURCE_KIND_REGEX_STRING)
}

// ValidateAPI returns one or more errors
// with a service.
func ValidateAPI(a *api.API) []error {
	errors := []error{}
	for _, r := range a.Resources {
		for _, err := range validateResource(r) {
			errors = append(errors, fmt.Errorf("error validating resource %q: %w", r.Singular, err))
		}
	}
	return errors
}

// validateResource returns any validation errors
// with a resource.
func validateResource(r *api.Resource) []error {
	regex := regexp.MustCompile(RESOURCE_KIND_REGEX_STRING)
	errors := []error{}
	if !regex.MatchString(r.Singular) {
		errors = append(
			errors,
			fmt.Errorf("kind must match regex %q", RESOURCE_KIND_REGEX_STRING),
		)
	}

	for _, p := range r.Schema.Properties {
		errors = append(errors, validateProperty(&p)...)
	}
	return errors
}

func validateProperty(p *openapi.Schema) []error {
	errors := []error{}
	if p.Ref != "" && p.Properties != nil {
		errors = append(errors, fmt.Errorf("cannot set both ref and properties on %v", p))
	}
	if p.Properties != nil {
		for _, p := range p.Properties {
			errors = append(errors, validateProperty(&p)...)
		}
	}
	return errors
}
