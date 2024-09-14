package openapi

import (
	"fmt"

	"github.com/aep-dev/aepc/schema"
)

type TypeInfo struct {
	 openapi_type string
	 openapi_format string
	 openapi_ref string

	 array_type *TypeInfo
}

func openAPIType(p *schema.Property) (TypeInfo, error) {
	if(p.Type == schema.Type_ARRAY) {
		at, err := openAPIType_helper(p.GetArrayPrimitiveType(), p.GetArrayObjectType())
		if(err != nil) {
			return TypeInfo{}, nil
		}
		return TypeInfo{
			openapi_type: "array",
			array_type: &at,
		}, nil
	}
	return openAPIType_helper(p.Type, p.ObjectType)
}

func openAPIType_helper(p schema.Type, object_type string) (TypeInfo, error) {
	t := "";
	f := "";
	r := "";

	switch(p) {
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
		case schema.Type_OBJECT:
			r = fmt.Sprintf("#/components/schemas/%s", object_type)
		default:
			return TypeInfo{}, fmt.Errorf("%s does not have openapi type support", p.Type)
	}

	return TypeInfo{
		openapi_type:   t,
		openapi_format: f,
		openapi_ref: r,
	}, nil
}
