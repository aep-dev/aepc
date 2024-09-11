// package parser converts the schema
// into a full-fledged struct that provides
// more functionality for discovering resource references, etc.
package parser

import (
	"fmt"
	"sort"
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

func (ps *ParsedService) GetShortName() string {
	before, _, _ := strings.Cut(ps.Name, ".")
	return before
}

type ParsedResource struct {
	*schema.Resource
	Type    string
	Parents []*ParsedResource

	IsResource bool
}

type ParsedProperty struct {
	*schema.Property
	Name string
}

func NewParsedService(s *schema.Service) (*ParsedService, error) {
	resourceByType, err := loadResourceByType(s)
	if err != nil {
		return nil, fmt.Errorf("unable to build service %q: %w", s, err)
	}
	err = loadObjectsByType(s.Objects, s, &resourceByType)
	if err != nil {
		return nil, fmt.Errorf("unable to build service objects %q: %w", s, err)
	}
	ps := ParsedService{
		Service:        s,
		ResourceByType: resourceByType,
	}
	return &ps, nil
}

func ParsedResourceForObject(r *schema.Object, s *schema.Service) *ParsedResource {
	t := fmt.Sprintf("%s/%s", s.Name, r.Kind)
	return &ParsedResource{
			Type: t,
			Resource: &schema.Resource{
				Kind: r.Kind,
				Properties: r.Properties,
			},
			IsResource: false,
		}
}

func loadObjectsByType(o []*schema.Object, s *schema.Service, m *map[string]*ParsedResource) (error) {
	for _, r := range o {
		t := fmt.Sprintf("%s/%s", s.Name, r.Kind)
		(*m)[t] = ParsedResourceForObject(r, s)
	}
	return nil
}

func loadResourceByType(s *schema.Service) (map[string]*ParsedResource, error) {
	resourceByType := map[string]*ParsedResource{}
	for _, r := range s.Resources {
		t := fmt.Sprintf("%s/%s", s.Name, r.Kind)
		resourceByType[t] = &ParsedResource{
			Resource: r,
			Type:     t,
			Parents:  []*ParsedResource{},
			IsResource: true,
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

func (pr *ParsedResource) GetPropertiesSortedByNumber() []*ParsedProperty {
	// to ensure idempotency of generators, fields are ordered by
	// field number
	parsedProperties := []*ParsedProperty{}
	for name, p := range pr.Properties {
		parsedProperties = append(parsedProperties, &ParsedProperty{
			Property: p,
			Name:     name,
		})
	}
	sort.Slice(parsedProperties, func(i, j int) bool {
		return parsedProperties[i].Number < parsedProperties[j].Number
	})
	return parsedProperties
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
