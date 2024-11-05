package openapi

import (
	"encoding/json"
	"fmt"

	"github.com/aep-dev/aepc/constants"
	"github.com/aep-dev/aepc/parser"
	"github.com/aep-dev/aepc/schema"
	"github.com/aep-dev/aepc/writer/writer_utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var lowerizer cases.Caser

func init() {
	lowerizer = cases.Lower(language.AmericanEnglish)
}

func WriteServiceToOpenAPI(ps *parser.ParsedService) ([]byte, error) {
	openAPI, err := convertToOpenAPI(ps)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(openAPI, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

func convertToOpenAPI(service *parser.ParsedService) (*OpenAPI, error) {
	paths := Paths{}
	components := Components{
		Schemas: Schemas{},
	}
	for _, r := range service.ResourceByType {
		d, err := resourceToSchema(r)
		if err != nil {
			return nil, err
		}
		if !r.IsResource {
			components.Schemas[r.Kind] = d
			continue
		}
		// if it is a resource, add paths
		parentPWPS := generateParentPatternsWithParams(r)
		// add an empty PathWithParam, if there are no parents.
		// This will add paths for the simple resource case.
		if len(*parentPWPS) == 0 {
			*parentPWPS = append(*parentPWPS, PathWithParams{
				Pattern: "", Params: []ParameterInfo{},
			})
		}
		patterns := []string{}
		schemaRef := fmt.Sprintf("#/components/schemas/%v", r.Kind)
		singular := r.Kind
		collection := writer_utils.CollectionName(r)
		// declare some commonly used objects, to be used later.
		bodyParam := RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: Schema{
						Ref: schemaRef,
					},
				},
			},
		}
		idParam := ParameterInfo{
			In:       "path",
			Name:     singular,
			Required: true,
			Type:     "string",
		}
		resourceResponse := ResponseInfo{
			Description: "Successful response",
			Content: map[string]MediaType{
				"application/json": {
					Schema: Schema{
						Ref: schemaRef,
					},
				},
			},
		}
		for _, pwp := range *parentPWPS {
			resourcePath := fmt.Sprintf("%s/%s/{%s}", pwp.Pattern, collection, singular)
			patterns = append(patterns, resourcePath)
			if r.Methods.List != nil {
				listPath := fmt.Sprintf("%s/%s", pwp.Pattern, collection)
				responseProperties := Properties{
					"results": Schema{
						Type: "array",
						Items: &Schema{
							Ref: schemaRef,
						},
					},
				}
				if r.Methods.List.UnreachableResources {
					responseProperties["unreachable"] = Schema{
						Type: "array",
						Items: &Schema{
							Type: "string",
						},
					}
				}
				addMethodToPath(paths, listPath, "get", MethodInfo{
					Parameters: append(pwp.Params,
						ParameterInfo{
							In:       "query",
							Name:     "max_page_size",
							Required: true,
							Type:     "integer",
						},
						ParameterInfo{
							In:       "query",
							Name:     "page_token",
							Required: true,
							Type:     "string",
						},
					),
					Responses: Responses{
						"200": ResponseInfo{
							Description: "Successful response",
							Content: map[string]MediaType{
								"application/json": {
									Schema: Schema{
										Type:       "object",
										Properties: &responseProperties,
									},
								},
							},
						},
					},
				})
			}
			if r.Methods.Create != nil {
				createPath := fmt.Sprintf("%s/%s", pwp.Pattern, collection)
				params := pwp.Params
				if !r.Methods.Create.NonClientSettableId {
					params = append(params, ParameterInfo{
						In:       "query",
						Name:     "id",
						Required: true,
						Type:     "string",
					})
				}
				addMethodToPath(paths, createPath, "post", MethodInfo{
					Parameters:  params,
					RequestBody: &bodyParam,
					Responses: Responses{
						"200": resourceResponse,
					},
				})
			}
			if r.Methods.Read != nil {
				addMethodToPath(paths, resourcePath, "get", MethodInfo{
					Parameters: append(pwp.Params, idParam),
					Responses: Responses{
						"200": resourceResponse,
					},
				})
			}
			if r.Methods.Update != nil {
				addMethodToPath(paths, resourcePath, "patch", MethodInfo{
					Parameters:  append(pwp.Params, idParam),
					RequestBody: &bodyParam,
					Responses: Responses{
						"200": resourceResponse,
					},
				})
			}
			if r.Methods.Delete != nil {
				addMethodToPath(paths, resourcePath, "delete", MethodInfo{
					Parameters: append(pwp.Params, idParam),
					Responses: Responses{
						"200": ResponseInfo{},
					},
				})
			}
			if r.Methods.Apply != nil {
				addMethodToPath(paths, resourcePath, "put", MethodInfo{
					Parameters:  append(pwp.Params, idParam),
					RequestBody: &bodyParam,
					Responses: Responses{
						"200": resourceResponse,
					},
				})
			}
		}
		d.XAEPResource = &XAEPResource{
			Singular: r.Kind,
			Plural:   r.Plural,
			Patterns: patterns,
			Parents:  r.Parents,
		}
		components.Schemas[r.Kind] = d
	}
	openAPI := &OpenAPI{
		OpenAPI: "3.1.0",
		Servers: []Server{
			{URL: service.Service.Url},
		},
		Info: Info{
			Title:   service.Service.Name,
			Version: "version not set",
		},
		Paths:      paths,
		Components: components,
	}
	return openAPI, nil
}

func buildProperties(props []*parser.ParsedProperty) (Properties, []string, error) {
	properties := Properties{}
	required := []string{}
	for _, f := range props {
		t, err := openAPIType(f.Property)
		if err != nil {
			return Properties{}, []string{}, err
		}
		s := Schema{
			Type:         t.openapi_type,
			Format:       t.openapi_format,
			Ref:          t.openapi_ref,
			XTerraformID: f.Name == constants.FIELD_ID_NAME,
			ReadOnly:     f.ReadOnly,
		}
		if f.Required {
			required = append(required, f.Name)
		}
		switch f.GetTypes().(type) {
		case *schema.Property_ArrayType:
			switch f.GetArrayType().GetArrayDetails().(type) {
			case *schema.ArrayType_ObjectType:
				s.Items = &Schema{
					Type:   t.array_type.openapi_type,
					Format: t.array_type.openapi_format,
				}
				if len(f.GetObjectType().GetProperties()) > 0 {
					props, required, err := buildProperties(parser.PropertiesSortedByNumber(f.GetObjectType().GetProperties()))
					if err != nil {
						return Properties{}, []string{}, err
					}
					s.Items.Required = required
					s.Items.Properties = &props
				}
			case *schema.ArrayType_Type:
				s.Items = &Schema{
					Type:   t.array_type.openapi_type,
					Format: t.array_type.openapi_format,
				}
			}
		case *schema.Property_ObjectType:
			if len(f.GetObjectType().GetProperties()) > 0 {
				props, required, err := buildProperties(parser.PropertiesSortedByNumber(f.GetObjectType().GetProperties()))
				if err != nil {
					return Properties{}, []string{}, err
				}
				s.Required = required
				s.Properties = &props
			}
		}
		properties[f.Name] = s
	}
	return properties, required, nil
}

func resourceToSchema(r *parser.ParsedResource) (Schema, error) {
	properties, required, err := buildProperties(r.GetPropertiesSortedByNumber())
	if err != nil {
		return Schema{}, err
	}
	return Schema{
		Type:       "object",
		Properties: &properties,
		Required:   required,
	}, nil
}

// PathWithParams passes an http path
// with the OpenAPI parameters it contains.
// helpful to bundle them both when iterating.
type PathWithParams struct {
	Pattern string
	Params  []ParameterInfo
}

// generate the x-aep-patterns for the parent resources, along with the patterns
// they need.
//
// This is helpful when you're constructing methods on resources with a parent.
func generateParentPatternsWithParams(r *parser.ParsedResource) *[]PathWithParams {
	if len(r.ParsedParents) == 0 {
		return &[]PathWithParams{}
	}
	pwps := []PathWithParams{}
	for _, parent := range r.ParsedParents {
		singular := parent.Kind
		basePattern := fmt.Sprintf("/%s/{%s}", writer_utils.CollectionName(parent), singular)
		baseParam := ParameterInfo{
			In:       "path",
			Name:     singular,
			Required: true,
			Type:     "string",
		}
		if len(parent.ParsedParents) == 0 {
			pwps = append(pwps, PathWithParams{
				Pattern: basePattern,
				Params:  []ParameterInfo{baseParam},
			})
		} else {
			for _, parentPWP := range *generateParentPatternsWithParams(parent) {
				params := append(parentPWP.Params, baseParam)
				pattern := fmt.Sprintf("%s%s", parentPWP.Pattern, basePattern)
				pwps = append(pwps, PathWithParams{Pattern: pattern, Params: params})
			}
		}
	}
	return &pwps
}

func addMethodToPath(paths Paths, path, method string, methodInfo MethodInfo) {
	methods, ok := paths[path]
	if !ok {
		methods = Methods{}
		paths[path] = methods
	}
	methods[method] = methodInfo
}

type OpenAPI struct {
	OpenAPI    string     `json:"openapi"`
	Servers    []Server   `json:"servers,omitempty"`
	Info       Info       `json:"info"`
	Paths      Paths      `json:"paths"`
	Components Components `json:"components"`
}

type Server struct {
	URL         string            `json:"url"`
	Description string            `json:"description,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Paths map[string]Methods

type Components struct {
	Schemas Schemas `json:"schemas"`
}

type Schemas map[string]Schema

type Methods map[string]MethodInfo

type MethodInfo struct {
	Responses   Responses    `json:"responses"`
	Parameters  Parameters   `json:"parameters,omitempty"`
	RequestBody *RequestBody `json:"requestBody,omitempty"`
}

type Responses map[string]ResponseInfo

type Parameters []ParameterInfo

type ResponseInfo struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content"`
}

type ParameterInfo struct {
	In       string `json:"in"`
	Name     string `json:"name"`
	Schema   Schema `json:"schema"`
	Required bool   `json:"required,omitempty"`
	Type     string `json:"type,omitempty"`
}

type Schema struct {
	Ref          string        `json:"$ref,omitempty"`
	Type         string        `json:"type,omitempty"`
	Format       string        `json:"format,omitempty"`
	Required     []string      `json:"required,omitempty"`
	ReadOnly     bool          `json:"readOnly,omitempty"`
	Items        *Schema       `json:"items,omitempty"`
	Properties   *Properties   `json:"properties,omitempty"`
	XTerraformID bool          `json:"x-terraform-id,omitempty"`
	XAEPResource *XAEPResource `json:"x-aep-resource,omitempty"`
}

type Properties map[string]Schema

type XAEPResource struct {
	Singular string   `json:"singular,omitempty"`
	Plural   string   `json:"plural,omitempty"`
	Patterns []string `json:"patterns,omitempty"`
	Parents  []string `json:"parents,omitempty"`
}

type RequestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema Schema `json:"schema"`
}
