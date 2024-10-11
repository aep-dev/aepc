package openapi

import (
	"encoding/json"
	"fmt"
	"log"

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
		components.Schemas[r.Kind] = d
		if !r.IsResource {
			continue
		}
		schemaRef := fmt.Sprintf("#/components/schemas/%v", r.Kind)
		if r.Methods.List != nil {
			log.Printf("resource plural: %s", r.Plural)
			listPath := fmt.Sprintf("/%s", lowerizer.String(r.Plural))
			addMethodToPath(paths, listPath, "get", MethodInfo{
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
			getPath := fmt.Sprintf("/%s", lowerizer.String(r.Plural))
			addMethodToPath(paths, getPath, "post", MethodInfo{
				Parameters: Parameters{
					ParameterInfo{
						In:   "body",
						Name: "body",
						Schema: Schema{
							Ref: schemaRef,
						},
					},
					ParameterInfo{
						In:       "path",
						Name:     "id",
						Required: true,
						Type:     "string",
					},
				},
				Responses: Responses{
					"200": ResponseInfo{
						Schema: Schema{
							Ref: schemaRef,
						},
					},
				},
			})
		}
		if r.Methods.Read != nil {
			getPath := fmt.Sprintf("/%s/{id}", lowerizer.String(r.Plural))
			addMethodToPath(paths, getPath, "get", MethodInfo{
				Responses: Responses{
					"200": ResponseInfo{
						Schema: Schema{
							Ref: schemaRef,
						},
					},
				},
			})
		}
		if r.Methods.Update != nil {
			getPath := fmt.Sprintf("/%s/{id}", lowerizer.String(r.Plural))
			addMethodToPath(paths, getPath, "patch", MethodInfo{
				Parameters: Parameters{
					ParameterInfo{
						In:   "body",
						Name: "body",
						Schema: Schema{
							Ref: schemaRef,
						},
					},
				},
				Responses: Responses{
					"200": ResponseInfo{
						Schema: Schema{
							Ref: schemaRef,
						},
					},
				},
			})
		}
		if r.Methods.Delete != nil {
			getPath := fmt.Sprintf("/%s/{id}", lowerizer.String(r.Plural))
			addMethodToPath(paths, getPath, "delete", MethodInfo{
				Responses: Responses{
					"200": ResponseInfo{
						Schema: Schema{
							Ref: schemaRef,
						},
					},
				},
			})
		}
		if r.Methods.Apply != nil {
			getPath := fmt.Sprintf("/%s/{id}", lowerizer.String(r.Plural))
			addMethodToPath(paths, getPath, "put", MethodInfo{
				Parameters: Parameters{
					ParameterInfo{
						In:   "body",
						Name: "body",
						Schema: Schema{
							Ref: schemaRef,
						},
					},
				},
				Responses: Responses{
					"200": ResponseInfo{
						Schema: Schema{
							Ref: schemaRef,
						},
					},
				},
			})
		}

	}
	openAPI := &OpenAPI{
		Swagger: "2.0",
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
		XAEPResource: &XAEPResource{
			Singular: r.Kind,
			Plural:   r.Plural,
			Patterns: []string{
				fmt.Sprintf("/%s/{%s}", lowerizer.String(r.Plural), lowerizer.String(r.Kind)),
			},
		},
	}, nil
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
	Info       Info       `json:"info"`
	Schemes    []string   `json:"schemes"`
	Paths      Paths      `json:"paths"`
	Components Components `json:"components"`
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
}
