package openapi

import (
	"encoding/json"
	"fmt"

	"github.com/aep-dev/aepc/constants"
	"github.com/aep-dev/aepc/parser"
	"github.com/aep-dev/aepc/schema"
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
		// declare some commonly used objects, to be used later.
		bodyParam := ParameterInfo{
			In:   "body",
			Name: "body",
			Schema: Schema{
				Ref: schemaRef,
			},
		}
		idParam := ParameterInfo{
			In:       "path",
			Name:     fmt.Sprintf("%s_id", lowerizer.String(r.Kind)),
			Required: true,
			Type:     "string",
		}
		resourceResponse := ResponseInfo{
			Schema: Schema{
				Ref: schemaRef,
			},
		}
		for _, pwp := range *parentPWPS {
			resourcePath := fmt.Sprintf("%s/%s/{%s_id}", pwp.Pattern, lowerizer.String(r.Plural), lowerizer.String(r.Kind))
			patterns = append(patterns, resourcePath)
			if r.Methods.List != nil {
				listPath := fmt.Sprintf("%s/%s", pwp.Pattern, lowerizer.String(r.Plural))
				addMethodToPath(paths, listPath, "get", MethodInfo{
					Parameters: pwp.Params,
					Responses: Responses{
						"200": ResponseInfo{
							Schema: Schema{
								Items: &Schema{
									Ref: schemaRef,
								},
							},
						},
					},
				})
			}
			if r.Methods.Create != nil {
				createPath := fmt.Sprintf("%s/%s", pwp.Pattern, lowerizer.String(r.Plural))
				params := append(pwp.Params, bodyParam,
					ParameterInfo{
						In:       "query",
						Name:     "id",
						Required: true,
						Type:     "string",
					},
				)
				addMethodToPath(paths, createPath, "post", MethodInfo{
					Parameters: params,
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
					Parameters: append(pwp.Params, idParam, bodyParam),
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
					Parameters: append(pwp.Params, bodyParam),
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
		Swagger: "2.0",
		Servers: []Server{
			{URL: service.Service.Url},
		},
		Info: Info{
			Title:   service.Service.Name,
			Version: "version not set",
		},
		Schemes:    []string{"http"},
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
		basePattern := fmt.Sprintf("/%s/{%s_id}", lowerizer.String(parent.Plural), lowerizer.String(parent.Kind))
		baseParam := ParameterInfo{
			In:       "path",
			Name:     fmt.Sprintf("%s_id", lowerizer.String(parent.Kind)),
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
				pattern := fmt.Sprintf("{%s}{%s}", parentPWP.Pattern, basePattern)
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
	Swagger    string     `json:"swagger"`
	Servers    []Server   `json:"servers,omitempty"`
	Info       Info       `json:"info"`
	Schemes    []string   `json:"schemes"`
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
	Responses  Responses  `json:"responses"`
	Parameters Parameters `json:"parameters"`
}

type Responses map[string]ResponseInfo

type Parameters []ParameterInfo

type ResponseInfo struct {
	Schema Schema `json:"schema"`
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
