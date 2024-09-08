package openapi

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/aep-dev/aepc/constants"
	"github.com/aep-dev/aepc/parser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var lowerizer cases.Caser

func init() {
	lowerizer = cases.Lower(language.AmericanEnglish)
}

func WriteServiceToOpenAPI(ps *parser.ParsedService) ([]byte, error) {
	openAPI, err := convertToOpenAPI(ps)
	if(err != nil) {
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
	definitions := Definitions{}
	for _, r := range service.ResourceByType {
		d, err := resourceToSchema(r)
		if(err != nil) {
			return nil, err
		}
		definitions[r.Kind] = d;
		schemaRef := fmt.Sprintf("#/definitions/%v", r.Kind)
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
		Schemes:     []string{"http"},
		Paths:       paths,
		Definitions: definitions,
	}
	return openAPI, nil
}

func resourceToSchema(r *parser.ParsedResource) (Schema, error) {
	properties := Properties{}
	required := []string{}
	for name, p := range r.Properties {
		t, err := openAPIType(p)
		if(err != nil ) {
			return Schema{}, err
		}
		properties[name] = Schema{
			Type:         t.openapi_type,
			Format:       t.openapi_format,
			XTerraformID: name == constants.FIELD_ID_NAME,
			ReadOnly:     p.ReadOnly,
		}
		if p.Required {
			required = append(required, name)
		}
	}
	return Schema{
		Type:       "object",
		Properties: &properties,
		Required:   required,
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
	Swagger     string      `json:"swagger"`
	Info        Info        `json:"info"`
	Schemes     []string    `json:"schemes"`
	Paths       Paths       `json:"paths"`
	Definitions Definitions `json:"definitions"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Paths map[string]Methods

type Definitions map[string]Schema

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
	Ref          string      `json:"$ref,omitempty"`
	Type         string      `json:"type,omitempty"`
	Format       string      `json:"format,omitempty"`
	Required     []string    `json:"required,omitempty"`
	ReadOnly     bool        `json:"readOnly,omitempty"`
	Items        *Schema     `json:"items,omitempty"`
	Properties   *Properties `json:"properties,omitempty"`
	XTerraformID bool        `json:"x-terraform-id,omitempty"`
}

type Properties map[string]Schema
