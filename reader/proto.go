package reader

import (
	"log"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/toumorokoshi/aep-sandbox/aepc/schema"
)

const resourcesFile = "resources.proto"

func ReadServiceFromProto(b []byte, s *schema.Service) error {
	// Create a new proto parser.
	accessor := protoparse.FileContentsFromMap(map[string]string{
		resourcesFile: string(b),
	})
	parser := protoparse.Parser{
		Accessor: accessor,
	}

	// Parse the proto file.
	files, err := parser.ParseFiles(resourcesFile)
	if err != nil {
		log.Fatal(err)
	}

	resources := []*schema.Resource{}

	for _, fd := range files {
		// find all services
		services := fd.GetServices()
		for _, protoS := range services {
			s.Name = protoS.GetName()
		}
		// find all messages
		messages := fd.GetMessageTypes()
		for _, m := range messages {
			r, err := MessageToResource(m)
			if err != nil {
				return nil
			}
			resources = append(resources, r)
		}
	}
	s.Resources = resources
	return nil
}

func MessageToResource(m *desc.MessageDescriptor) (*schema.Resource, error) {
	r := &schema.Resource{
		Kind: m.GetName(),
	}
	return r, nil
}
