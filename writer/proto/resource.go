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
package proto

import (
	"fmt"
	"strings"

	"github.com/aep-dev/aepc/schema"
	"github.com/jhump/protoreflect/desc/builder"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// AddResource adds a resource's protos and RPCs to a file and service.
func AddResource(r *schema.Resource, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	resourceMb, err := GeneratedResourceMessage(r)
	if err != nil {
		return fmt.Errorf("unable to generated resource %v: %w", r.Kind, err)
	}
	fb.AddMessage(resourceMb)
	err = AddCreate(r, resourceMb, fb, sb)
	if err != nil {
		return err
	}
	err = AddRead(r, resourceMb, fb, sb)
	if err != nil {
		return err
	}
	// err = AddDelete(r, resourceMb, fb, sb)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// GenerateResourceMesssage adds the resource message.
func GeneratedResourceMessage(r *schema.Resource) (*builder.MessageBuilder, error) {
	mb := builder.NewMessage(r.Kind)
	mb.AddField(
		builder.NewField(FIELD_NAME_PATH, builder.FieldTypeString()).SetNumber(1),
	)
	mb.SetOptions(
		&descriptorpb.MessageOptions{},
		// annotations.ResourceDescriptor{
		//	"type": sb.GetName() + "/" + r.Kind,
		//},
	)
	// md.GetMessageOptions().ProtoReflect().Set(protoreflect.FieldDescriptor, protoreflect.Value)
	// mb.AddNestedExtension(
	// 	builder.NewExtension("google.api.http", tag int32, typ *builder.FieldType, extendee *builder.MessageBuilder)
	// )
	return mb, nil
}

func AddCreate(r *schema.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Create" + r.Kind + "Request")
	mb.AddField(builder.NewField(FIELD_NAME_ID, builder.FieldTypeString()).SetNumber(1))
	mb.AddField(builder.NewField(FIELD_NAME_RESOURCE, builder.FieldTypeMessage(resourceMb)).SetNumber(2))
	fb.AddMessage(mb)
	method := builder.NewMethod("Create"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Post{
			Post: "/" + strings.ToLower(r.Kind),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddRead(r *schema.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Read" + r.Kind + "Request")
	mb.AddField(
		builder.NewField(FIELD_NAME_PATH, builder.FieldTypeString()).SetNumber(1),
	)
	fb.AddMessage(mb)
	method := builder.NewMethod("Read"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: fmt.Sprintf("/{path=%s/*}", strings.ToLower(r.Kind)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddDelete(r *schema.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Delete" + r.Kind + "Request")
	mb.AddField(
		builder.NewField(FIELD_NAME_PATH, builder.FieldTypeString()).SetNumber(1),
	)
	fb.AddMessage(mb)
	method := builder.NewMethod("Delete"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(nil, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Delete{
			Delete: fmt.Sprintf("/{path=%s/*}", strings.ToLower(r.Kind)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}
