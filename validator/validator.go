// Package validator exposes validation functionality for the
// resource definition.
package validator

import (
	"fmt"
	"regexp"

	"github.com/aep-dev/aepc/schema"
)

const (
	RESOURCE_KIND_REGEX_STRING = "[A-Z]+[a-zA-Z0-9]*"
)

var RESOURCE_KIND_REGEX regexp.Regexp

func init() {
	// compile regex via init rather than const to use the other string
	// constants.
	RESOURCE_KIND_REGEX = *regexp.MustCompile(RESOURCE_KIND_REGEX_STRING)
}

// ValidateService returns one or more errors
// with a service.
func ValidateService(s *schema.Service) []error {
	errors := []error{}
	for _, r := range s.Resources {
		for _, err := range validateResource(r) {
			errors = append(errors, fmt.Errorf("error validating resource %q: %w", r.Kind, err))
		}
	}
	return errors
}

// validateResource returns any validation errors
// with a resource.
func validateResource(r *schema.Resource) []error {
	regex := regexp.MustCompile(RESOURCE_KIND_REGEX_STRING)
	errors := []error{}
	if !regex.MatchString(r.Kind) {
		errors = append(
			errors,
			fmt.Errorf("kind must match regex %q", RESOURCE_KIND_REGEX_STRING),
		)
	}

	for _, p := range r.Properties {
		errors = append(errors, validateProperty(p)...)
	}
	return errors
}

func validateProperty(p *schema.Property) []error {
	errors := []error{}
	switch p.GetTypes().(type) {
	case *schema.Property_ObjectType:
		if p.GetObjectType().GetMessageName() != "" && len(p.GetObjectType().GetProperties()) != 0 {
			errors = append(errors, fmt.Errorf("cannot set both message_name and properties on object_type %q", p))
		}
		for _, p := range p.GetObjectType().GetProperties() {
			errors = append(errors, validateProperty(p)...)
		}
	}
	return errors
}
