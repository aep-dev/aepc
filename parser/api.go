package parser

import (
	"fmt"

	"github.com/aep-dev/aep-lib-go/pkg/api"
	"github.com/aep-dev/aep-lib-go/pkg/openapi"
	"github.com/aep-dev/aepc/schema"
)

func ToAPI(s *schema.Service) (*api.API, error) {
	schemas := make(map[string]*openapi.Schema)
	resourceByName := make(map[string]*schema.Resource)
	for _, r := range s.Resources {
		resourceByName[r.Kind] = r
	}
	resources := map[string]*api.Resource{}
	for _, r := range s.Resources {
		_, err := getOrCreateResource(resources, resourceByName, r.Kind)
		if err != nil {
			return nil, err
		}
	}
	return &api.API{
		ServerURL: s.Url,
		Name:      s.Name,
		Schemas:   schemas,
		Resources: resources,
	}, nil
}

func getOrCreateResource(apiResourceByName map[string]*api.Resource, resourceByName map[string]*schema.Resource, name string) (*api.Resource, error) {
	if apiR, ok := apiResourceByName[name]; ok {
		return apiR, nil
	}
	schemaR, ok := resourceByName[name]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", name)
	}
	schema, err := toOpenAPISchemaFromPropMap(schemaR.Properties)
	if err != nil {
		return nil, err
	}
	parents := []*api.Resource{}
	apiR := &api.Resource{
		Singular: schemaR.Kind,
		Plural:   schemaR.Plural,
		Parents:  parents,
		Schema:   schema,
	}
	methods := schemaR.GetMethods()
	if methods.Read != nil {
		apiR.GetMethod = &api.GetMethod{}
	}
	if methods.Create != nil {
		apiR.CreateMethod = &api.CreateMethod{
			SupportsUserSettableCreate: !methods.Create.GetNonClientSettableId(),
		}
	}
	if methods.Update != nil {
		apiR.UpdateMethod = &api.UpdateMethod{}
	}
	if methods.Delete != nil {
		apiR.DeleteMethod = &api.DeleteMethod{}
	}
	if methods.List != nil {
		apiR.ListMethod = &api.ListMethod{}
	}
	for _, p := range schemaR.Parents {
		apiP, err := getOrCreateResource(apiResourceByName, resourceByName, p)
		if err != nil {
			return nil, err
		}
		apiP.Children = append(apiP.Children, apiR)
		apiR.Parents = append(apiR.Parents, apiP)
	}
	apiResourceByName[name] = apiR
	return apiR, nil
}
