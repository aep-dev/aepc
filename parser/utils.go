package parser

import (
	"fmt"

	"github.com/aep-dev/aep-lib-go/pkg/openapi"
	"github.com/aep-dev/aepc/schema"
)

func toOpenAPISchema(p *schema.Property) (*openapi.Schema, error) {
	switch p.GetTypes().(type) {
	case *schema.Property_ArrayType:
		return toOpenAPIArray(p.GetArrayType())
	case *schema.Property_ObjectType:
		return openAPITypeObject(p.GetObjectType())
	case *schema.Property_Type:
		return openAPITypePrimitive(p.GetType())
	default:
		return nil, fmt.Errorf("openapi type for %q not found", p.GetTypes())
	}
}

func toOpenAPIArray(a *schema.ArrayType) (*openapi.Schema, error) {
	var itemType *openapi.Schema
	switch a.GetArrayDetails().(type) {
	case *schema.ArrayType_Type:
		at, err := openAPITypePrimitive(a.GetType())
		if err != nil {
			return nil, err
		}
		itemType = at
	case *schema.ArrayType_ObjectType:
		ot, err := openAPITypeObject(a.GetObjectType())
		if err != nil {
			return nil, err
		}
		itemType = ot
	}

	return &openapi.Schema{
		Type:  "array",
		Items: itemType,
	}, nil
}

func openAPITypeObject(o *schema.ObjectType) (*openapi.Schema, error) {
	if o.GetMessageName() != "" {
		return &openapi.Schema{
			Ref: fmt.Sprintf("#/components/schemas/%s", o.GetMessageName()),
		}, nil
	} else {
		return toOpenAPISchemaFromPropMap(o.GetProperties())
	}
}

func toOpenAPISchemaFromPropMap(propMap map[string]*schema.Property) (*openapi.Schema, error) {
	required := []string{}
	properties := openapi.Properties{}
	field_numbers := map[int]string{}
	for name, p := range propMap {
		prop, err := toOpenAPISchema(p)
		if err != nil {
			return nil, err
		}
		properties[name] = *prop
		if p.GetRequired() {
			required = append(required, name)
		}
		field_numbers[int(p.GetNumber())] = name
	}
	return &openapi.Schema{
		Type:             "object",
		Properties:       properties,
		Required:         required,
		XAEPFieldNumbers: field_numbers,
	}, nil
}

func openAPITypePrimitive(p schema.Type) (*openapi.Schema, error) {
	t := ""
	f := ""

	switch p {
	case schema.Type_STRING:
		t = "string"
	case schema.Type_DOUBLE:
		t = "number"
		f = "double"
	case schema.Type_FLOAT:
		t = "number"
		f = "float"
	case schema.Type_INT32:
		t = "integer"
		f = "int32"
	case schema.Type_INT64:
		t = "integer"
		f = "int64"
	case schema.Type_BOOLEAN:
		t = "boolean"
	default:
		return nil, fmt.Errorf("%s does not have openapi type support", p)
	}

	return &openapi.Schema{
		Type:   t,
		Format: f,
	}, nil
}
