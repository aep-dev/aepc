package openapi

import (
	"fmt"

	"github.com/aep-dev/aepc/schema"
)

type TypeInfo struct {
	 openapi_type string
	 openapi_format string
}

func openAPIType(p *schema.Property) (TypeInfo, error) {
	t := "";
	f := "";

	switch(p.Type) {
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
		openapi_type: t,
		openapi_format: f,
	}, nil
}