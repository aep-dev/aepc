package openapi

import "github.com/aep-dev/aepc/schema"

type TypeInfo struct {
	 openapi_type string
	 openapi_format string
}

func openAPIType(p *schema.Property) TypeInfo {
	t := "";
	f := "";

	switch(p.Type) {
		case schema.Type_STRING:
			t = "string"
	}

	return TypeInfo{
		openapi_type: t,
		openapi_format: f,
	}
}