package openapi

import (
	"fmt"

	"github.com/aep-dev/aepc/schema"
)

type TypeInfo struct {
	openapi_type   string
	openapi_format string
	openapi_ref    string

	array_type *TypeInfo
}

func openAPIType(p *schema.Property) (TypeInfo, error) {
	switch p.GetTypes().(type) {
	case *schema.Property_ArrayType:
		return openAPITypeArray(p.GetArrayType())
	case *schema.Property_ObjectType:
		return openAPITypeObject(p.GetObjectType())
	case *schema.Property_Type:
		return openAPITypePrimitive(p.GetType())
	default:
		return TypeInfo{}, fmt.Errorf("openapi type for %q not found", p.GetTypes())
	}
}

func openAPITypeArray(a *schema.ArrayType) (TypeInfo, error) {
	switch a.GetArrayDetails().(type) {
	case *schema.ArrayType_Type:
		at, err := openAPITypePrimitive(a.GetType())
		if err != nil {
			return TypeInfo{}, err
		}
		return TypeInfo{
			openapi_type: "array",
			array_type:   &at,
		}, nil
	case *schema.ArrayType_ObjectType:
		ot, err := openAPITypeObject(a.GetObjectType())
		if err != nil {
			return TypeInfo{}, err
		}
		return TypeInfo{
			openapi_type: "array",
			array_type:   &ot,
		}, nil

	default:
		return TypeInfo{}, fmt.Errorf("OpenAPI type for %q not found", a.GetArrayDetails())
	}
}

func openAPITypeObject(o *schema.ObjectType) (TypeInfo, error) {
	if o.GetMessageName() != "" {
		return TypeInfo{
			openapi_ref: fmt.Sprintf("#/components/schemas/%s", o.GetMessageName()),
		}, nil
	} else {
		return TypeInfo{openapi_type: "object"}, nil
	}
}

func openAPITypePrimitive(p schema.Type) (TypeInfo, error) {
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
		return TypeInfo{}, fmt.Errorf("%s does not have openapi type support", p.Type)
	}

	return TypeInfo{
		openapi_type:   t,
		openapi_format: f,
	}, nil
}
