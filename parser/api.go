package parser

import (
	"fmt"
	"strings"

	"github.com/aep-dev/aep-lib-go/pkg/api"
	"github.com/aep-dev/aep-lib-go/pkg/constants"
	"github.com/aep-dev/aep-lib-go/pkg/openapi"
	"github.com/aep-dev/aepc/schema"
)

func ToAPI(s *schema.Service) (*api.API, error) {
	schemas := make(map[string]*openapi.Schema)
	resourceByName := make(map[string]*schema.Resource)
	for _, r := range s.Resources {
		resourceByName[r.Kind] = r
	}
	for _, s := range s.Schemas {
		oasSchema, err := toOpenAPISchemaFromPropMap(s.Properties)
		if err != nil {
			return nil, err
		}
		schemas[s.Name] = oasSchema
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
	oasSchema, err := toOpenAPISchemaFromPropMap(schemaR.Properties)
	if err != nil {
		return nil, err
	}
	addCommonFieldsToResourceSchema(oasSchema)
	parents := []*api.Resource{}
	apiR := &api.Resource{
		Singular: schemaR.Kind,
		Plural:   schemaR.Plural,
		Parents:  parents,
		Schema:   oasSchema,
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
		apiR.ListMethod = &api.ListMethod{
			HasUnreachableResources: methods.List.GetHasUnreachableResources(),
		}
	}
	if methods.Apply != nil {
		apiR.ApplyMethod = &api.ApplyMethod{}
	}
	for _, cm := range methods.Custom {
		request, err := toOpenAPISchema(cm.Request)
		if err != nil {
			return nil, err
		}
		response, err := toOpenAPISchema(cm.Response)
		if err != nil {
			return nil, err
		}
		method := "POST"
		switch cm.MethodType {
		case schema.Methods_CustomMethod_GET:
			method = "GET"
		case schema.Methods_CustomMethod_POST:
			method = "POST"
		}
		apiR.CustomMethods = append(apiR.CustomMethods, &api.CustomMethod{
			Name:     cm.Name,
			Method:   method,
			Request:  request,
			Response: response,
		})
	}
	for _, p := range schemaR.Parents {
		apiP, err := getOrCreateResource(apiResourceByName, resourceByName, p)
		if err != nil {
			return nil, err
		}
		apiP.Children = append(apiP.Children, apiR)
		apiR.Parents = append(apiR.Parents, apiP)
	}
	// Generate pattern elements for the resource
	apiR.PatternElems = strings.Split(api.GeneratePatternStrings(apiR)[0], "/")
	apiResourceByName[name] = apiR
	return apiR, nil
}

// add an id field to the resource.
func addCommonFieldsToResourceSchema(s *openapi.Schema) {
	s.Properties[constants.FIELD_PATH_NAME] = openapi.Schema{
		Type:     "string",
		ReadOnly: true,
	}
	s.XAEPFieldNumbers[constants.FIELD_PATH_NUMBER] = constants.FIELD_PATH_NAME
}
