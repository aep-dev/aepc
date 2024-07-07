// package parser converts the schema
// into a full-fledged struct that provides
// more functionality for discovering resource references, etc.
package parser

import (
	"fmt"
	"strings"

	"github.com/aep-dev/aepc/constants"
	"github.com/aep-dev/aepc/schema"
)

// ParsedService wraps schema.Service, but includes
// helper functions for things like retrieving the resource
// definitions within a service.
type ParsedService struct {
	*schema.Service
	ResourceByType map[string]*ParsedResource
}

type ParsedResource struct {
	*schema.Resource
	Type    string
	Parents []*ParsedResource
}

func NewParsedService(s *schema.Service) (*ParsedService, error) {
	resourceByType, err := loadResourceByType(s)
	if err != nil {
		return nil, fmt.Errorf("unable to build service %q: %w", s, err)
	}
	ps := ParsedService{
		Service:        s,
		ResourceByType: resourceByType,
	}
	return &ps, nil
}

func loadResourceByType(s *schema.Service) (map[string]*ParsedResource, error) {
	resourceByType := map[string]*ParsedResource{}
	for _, r := range s.Resources {
		t := fmt.Sprintf("%s/%s", s.Name, r.Kind)
		resourceByType[t] = &ParsedResource{
			Resource: r,
			Type:     t,
			Parents:  []*ParsedResource{},
		}
	}
	// populate resource parents
	for _, r := range resourceByType {
		for _, p := range r.Resource.Parents {
			// if the string is a shorthand resource type (sans service),
			// build it before checking it's existence.
			if !strings.Contains(p, "/") {
				p = strings.Join([]string{s.Name, p}, "/")
			}
			parentResource, exists := resourceByType[p]
			if !exists {
				return nil, fmt.Errorf("parent %q for resource %q not found", p, r.Kind)
			}
			r.Parents = append(r.Parents, parentResource)
		}
		addGetToResource(r)
		addCommonFieldsToResource(r)
	}
	return resourceByType, nil
}

// addGetToResource adds a Get method to a resource,
// since all resources must have a Get method.
func addGetToResource(pr *ParsedResource) {
	if pr.Methods.Read == nil {
		pr.Methods.Read = &schema.Methods_ReadMethod{}
	}
}

// add an id field to the resource.
// TODO(yft): this has to be reconciled with the
// existence of path.
func addCommonFieldsToResource(pr *ParsedResource) {
	pr.Properties[constants.FIELD_PATH_NAME] = &schema.Property{
		Type:     schema.Type_STRING,
		Number:   10000,
		ReadOnly: true,
	}
	pr.Properties[constants.FIELD_ID_NAME] = &schema.Property{
		Type:     schema.Type_STRING,
		Number:   10001,
		ReadOnly: true,
	}
}
