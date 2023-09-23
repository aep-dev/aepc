// package parser converts the schema
// into a full-fledged struct that provides
// more functionality for discovering resource references, etc.
package parser

import (
	"fmt"

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
		name := fmt.Sprintf("%s/%s", s.Name, r.Kind)
		resourceByType[name] = &ParsedResource{
			Resource: r,
			Parents:  []*ParsedResource{},
		}
	}
	// populate resource parents
	for _, r := range resourceByType {
		for _, p := range r.Resource.Parents {
			parentResource, exists := resourceByType[p]
			if !exists {
				return nil, fmt.Errorf("parent %q for resource %q not found", p, r.Kind)
			}
			r.Parents = append(r.Parents, parentResource)
		}
	}
	return resourceByType, nil
}
