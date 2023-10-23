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

	"github.com/aep-dev/aepc/parser"
	"github.com/aep-dev/aepc/schema"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// AddResource adds a resource's protos and RPCs to a file and service.
func AddResource(r *parser.ParsedResource, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	resourceMb, err := GeneratedResourceMessage(r)
	if err != nil {
		return fmt.Errorf("unable to generated resource %v: %w", r.Kind, err)
	}
	fb.AddMessage(resourceMb)
	if r.Methods != nil {
		if r.Methods.Create != nil {
			err = AddCreate(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Read != nil {
			err = AddRead(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Update != nil {
			err = AddUpdate(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Delete != nil {
			err = AddDelete(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.List != nil {
			err = AddList(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GenerateResourceMesssage adds the resource message.
func GeneratedResourceMessage(r *parser.ParsedResource) (*builder.MessageBuilder, error) {
	mb := builder.NewMessage(r.Kind)
	// standard fields start at 10k, in the range until 11k.
	mb.AddField(
		builder.NewField(FIELD_NAME_PATH, builder.FieldTypeString()).SetNumber(10000),
	)
	// standard fields are added afterward.
	for n, p := range r.Properties {
		typ := builder.FieldTypeBool()
		switch p.Type {
		case schema.Type_STRING:
			typ = builder.FieldTypeString()
		}
		mb.AddField(builder.NewField(n, typ).SetNumber(p.Number))
	}
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

func AddCreate(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Create" + r.Kind + "Request")
	mb.AddField(builder.NewField(FIELD_NAME_PARENT, builder.FieldTypeString()).SetNumber(1))
	mb.AddField(builder.NewField(FIELD_NAME_ID, builder.FieldTypeString()).SetNumber(2))
	mb.AddField(builder.NewField(FIELD_NAME_RESOURCE, builder.FieldTypeMessage(resourceMb)).SetNumber(3))
	fb.AddMessage(mb)
	method := builder.NewMethod("Create"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Post{
			Post: generateParentHTTPPath(r),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddRead adds a read method for the resource, along with
// any required messages.
func AddRead(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
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
			Get: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddRead adds a read method for the resource, along with
// any required messages.
func AddUpdate(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Update" + r.Kind + "Request")
	mb.AddField(
		builder.NewField(FIELD_NAME_PATH, builder.FieldTypeString()).SetNumber(1),
	).AddField(
		builder.NewField(FIELD_NAME_RESOURCE, builder.FieldTypeMessage(resourceMb)).SetNumber(2),
	)
	fb.AddMessage(mb)
	method := builder.NewMethod("Update"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: fmt.Sprintf("/{resource.path=%v}", generateHTTPPath(r)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddDelete(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Delete" + r.Kind + "Request")
	mb.AddField(
		builder.NewField(FIELD_NAME_PATH, builder.FieldTypeString()).SetNumber(1),
	)
	fb.AddMessage(mb)
	emptyMd, err := desc.LoadMessageDescriptor("google.protobuf.Empty")
	if err != nil {
		return err
	}
	method := builder.NewMethod("Delete"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeImportedMessage(emptyMd, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Delete{
			Delete: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddList(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	reqMb := builder.NewMessage("List" + r.Kind + "Request")
	reqMb.AddField(
		builder.NewField(FIELD_NAME_PARENT, builder.FieldTypeString()).SetNumber(1),
	)
	fb.AddMessage(reqMb)
	respMb := builder.NewMessage("List" + r.Kind + "Response")
	respMb.AddField(
		builder.NewField(FIELD_NAME_RESOURCES, builder.FieldTypeMessage(resourceMb)).SetRepeated().SetNumber(1),
	)
	fb.AddMessage(respMb)
	method := builder.NewMethod("List"+r.Kind,
		builder.RpcTypeMessage(reqMb, false),
		builder.RpcTypeMessage(respMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: generateParentHTTPPath(r),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func generateHTTPPath(r *parser.ParsedResource) string {
	elements := []string{strings.ToLower(r.Kind)}
	if len(r.Parents) > 0 {
		// TODO: handle multiple parents
		p := r.Parents[0]
		for p != nil {
			elements = append([]string{strings.ToLower(p.Kind)}, elements...)
			if len(p.Parents) == 0 {
				break
			}
		}
	}
	return fmt.Sprintf("%v/*", strings.Join(elements, "/*/"))
}

func generateParentHTTPPath(r *parser.ParsedResource) string {
	parentPath := ""
	if len(r.Parents) > 0 {
		parentPath = fmt.Sprintf("{parent=%v}/", generateHTTPPath(r.Parents[0]))
	}
	return fmt.Sprintf("/%v%v", parentPath, strings.ToLower(r.Kind))
}
