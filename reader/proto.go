// Copyright 2023 Yusuke Fredrick Tsutsumi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package reader

import (
	"log"

	"github.com/aep-dev/aepc/schema"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
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
