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
	"reflect"
	"strings"

	"github.com/aep-dev/aepc/constants"
	"github.com/aep-dev/aepc/parser"
	"github.com/aep-dev/aepc/schema"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// AddResource adds a resource's protos and RPCs to a file and service.
func AddResource(r *parser.ParsedResource, ps *parser.ParsedService, fb *builder.FileBuilder, sb *builder.ServiceBuilder, m *MessageStorage) error {
	// Do not recreate resources if they've already been created.
	resourceMb, ok := m.Messages[fmt.Sprintf("%s/%s", ps.Name, r.Kind)]
	if !ok {
		return fmt.Errorf("%s not found in message storage", r.Kind)
	}

	fb.AddMessage(resourceMb)

	if !r.IsResource {
		return nil
	}

	if r.Methods != nil {
		if r.Methods.Create != nil {
			err := AddCreate(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Read != nil {
			err := AddGet(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Update != nil {
			err := AddUpdate(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.Delete != nil {
			err := AddDelete(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.List != nil {
			err := AddList(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
		if r.Methods.GlobalList != nil {
			err := AddGlobalList(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}

		if r.Methods.Apply != nil {
			err := AddApply(r, resourceMb, fb, sb)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func protoType(p *parser.ParsedProperty, s *parser.ParsedService, m *MessageStorage, parent *builder.MessageBuilder) (*builder.FieldType, error) {
	switch p.GetTypes().(type) {
	case *schema.Property_Type:
		return protoTypePrimitive(p.GetType())
	case *schema.Property_ObjectType:
		return protoTypeObject(p.GetObjectType(), p, s, m, parent)
	case *schema.Property_ArrayType:
		return protoTypeArray(p.GetArrayType(), p, s, m, parent)
	default:
		return nil, fmt.Errorf("reached outside of prototype switch statement.")
	}
}

func protoTypeObject(o *schema.ObjectType, p *parser.ParsedProperty, s *parser.ParsedService, m *MessageStorage, parent *builder.MessageBuilder) (*builder.FieldType, error) {
	if o.GetMessageName() != "" {
		wantedType := fmt.Sprintf("%s/%s", s.Name, o.GetMessageName())
		_, ok := m.Messages[wantedType]
		if !ok {
			// Resource has not been generated yet.
			n, ok := s.ResourceByType[wantedType]
			if !ok {
				return nil, fmt.Errorf("could not find %s in full object list", wantedType)
			}
			_, err := GeneratedResourceMessage(n, s, m)
			if err != nil {
				return nil, err
			}
		}
		resourceMb, ok := m.Messages[wantedType]
		if !ok {
			return nil, fmt.Errorf("could not find message %s after recursive create", wantedType)
		}
		return builder.FieldTypeMessage(resourceMb), nil
	} else {
		msg, err := GenerateMessage(parser.PropertiesSortedByNumber(o.GetProperties()), toMessageName(p.Name), s, m)
		if err != nil {
			return nil, err
		}
		parent.AddNestedMessage(msg)
		return builder.FieldTypeMessage(msg), nil
	}
}

func protoTypeArray(a *schema.ArrayType, p *parser.ParsedProperty, s *parser.ParsedService, m *MessageStorage, parent *builder.MessageBuilder) (*builder.FieldType, error) {
	switch a.GetArrayDetails().(type) {
	case *schema.ArrayType_Type:
		// Repeated will be set later on.
		return protoTypePrimitive(a.GetType())
	case *schema.ArrayType_ObjectType:
		return protoTypeObject(a.GetObjectType(), p, s, m, parent)
	default:
		return nil, fmt.Errorf("Proto type for %q not found ", a)
	}
}

func protoTypePrimitive(t schema.Type) (*builder.FieldType, error) {
	switch t {
	case schema.Type_STRING:
		return builder.FieldTypeString(), nil
	case schema.Type_INT32:
		return builder.FieldTypeInt32(), nil
	case schema.Type_INT64:
		return builder.FieldTypeInt64(), nil
	case schema.Type_BOOLEAN:
		return builder.FieldTypeBool(), nil
	case schema.Type_DOUBLE:
		return builder.FieldTypeDouble(), nil
	case schema.Type_FLOAT:
		return builder.FieldTypeFloat(), nil
	default:
		return nil, fmt.Errorf("Proto type for %q not found", t)
	}
}

func protoField(p *parser.ParsedProperty, s *parser.ParsedService, m *MessageStorage, parent *builder.MessageBuilder) (*builder.FieldBuilder, error) {
	typ, err := protoType(p, s, m, parent)
	if err != nil {
		return nil, err
	}
	f := builder.NewField(p.Name, typ).SetNumber(p.Number).SetComments(
		builder.Comments{
			LeadingComment: fmt.Sprintf("Field for %v.", p.Name),
		},
	)
	switch p.GetTypes().(type) {
	case *schema.Property_ArrayType:
		f.SetRepeated()
	}
	o := &descriptorpb.FieldOptions{}
	if p.Required {
		proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	}
	f.SetOptions(o)
	return f, nil
}

func GenerateMessage(properties []*parser.ParsedProperty, name string, s *parser.ParsedService, m *MessageStorage) (*builder.MessageBuilder, error) {
	mb := builder.NewMessage(name)
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A %v.", name),
	})
	for _, p := range properties {
		f, err := protoField(p, s, m, mb)
		if err != nil {
			return nil, err
		}
		mb.AddField(f)
	}
	return mb, nil
}

// GenerateResourceMesssage adds the resource message.
func GeneratedResourceMessage(r *parser.ParsedResource, s *parser.ParsedService, m *MessageStorage) (*builder.MessageBuilder, error) {
	mb, err := GenerateMessage(r.GetPropertiesSortedByNumber(), r.Kind, s, m)
	if err != nil {
		return nil, err
	}
	m.Messages[fmt.Sprintf("%s/%s", s.Name, r.Kind)] = mb
	return mb, nil
}

func AddCreate(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Create" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A Create request for a  %v resource.", r.Kind),
	})
	addParentField(r, mb)
	addIdField(r, mb)
	addResourceField(r, resourceMb, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Create"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Create method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Post{
			// TODO(yft): switch this over to use "id" in the path.
			Post: generateParentHTTPPath(r),
		},
		Body: strings.ToLower(r.Kind),
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PARENT_NAME, strings.ToLower(r.Kind)}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddGet adds a read method for the resource, along with
// any required messages.
func AddGet(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Get" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Get%v method", r.Kind),
	})
	addPathField(r, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Get"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Get method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PATH_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddRead adds a read method for the resource, along with
// any required messages.
func AddUpdate(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Update" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Update%v method", r.Kind),
	})
	addPathField(r, mb)
	addResourceField(r, resourceMb, mb)
	// TODO: find a way to get the actual field mask proto descriptor type, without
	// querying the global registry.
	fieldMaskDescriptor, _ := desc.LoadMessageDescriptorForType(reflect.TypeOf(fieldmaskpb.FieldMask{}))
	mb.AddField(builder.NewField(constants.FIELD_UPDATE_MASK_NAME, builder.FieldTypeImportedMessage(fieldMaskDescriptor)).
		SetNumber(constants.FIELD_UPDATE_MASK_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: "The update mask for the resource",
		}))

	fb.AddMessage(mb)
	method := builder.NewMethod("Update"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Update method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Patch{
			Patch: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
		Body: strings.ToLower(r.Kind),
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{strings.ToLower(r.Kind), constants.FIELD_UPDATE_MASK_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddDelete(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Delete" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Delete%v method", r.Kind),
	})
	addPathField(r, mb)
	fb.AddMessage(mb)
	emptyMd, err := desc.LoadMessageDescriptor("google.protobuf.Empty")
	if err != nil {
		return err
	}
	method := builder.NewMethod("Delete"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeImportedMessage(emptyMd, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Delete method for %v.", r.Kind),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Delete{
			Delete: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PATH_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddList(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	reqMb := builder.NewMessage("List" + r.Kind + "Request")
	reqMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the List%v method", r.Kind),
	})
	addParentField(r, reqMb)
	addPageToken(r, reqMb)
	reqMb.AddField(builder.NewField(constants.FIELD_MAX_PAGE_SIZE_NAME, builder.FieldTypeInt32()).
		SetNumber(constants.FIELD_MAX_PAGE_SIZE_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The maximum number of resources to return in a single page."),
		}))
	fb.AddMessage(reqMb)
	respMb := builder.NewMessage("List" + r.Kind + "Response")
	respMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Response message for the List%v method", r.Kind),
	})
	addResourcesField(r, resourceMb, respMb)
	addNextPageToken(r, respMb)
	fb.AddMessage(respMb)
	method := builder.NewMethod("List"+r.Kind,
		builder.RpcTypeMessage(reqMb, false),
		builder.RpcTypeMessage(respMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant List method for %v.", r.Plural),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: generateParentHTTPPath(r),
		},
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PARENT_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddGlobalList(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	reqMb := builder.NewMessage("GlobalList" + r.Kind + "Request")
	reqMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the GlobalList%v method", r.Kind),
	})
	addPathField(r, reqMb)
	addPageToken(r, reqMb)
	fb.AddMessage(reqMb)
	respMb := builder.NewMessage("GlobalList" + r.Kind + "Response")
	respMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Response message for the GlobalList%v method", r.Kind),
	})
	addResourcesField(r, resourceMb, respMb)
	addNextPageToken(r, respMb)
	fb.AddMessage(respMb)
	method := builder.NewMethod("GlobalList"+r.Kind,
		builder.RpcTypeMessage(reqMb, false),
		builder.RpcTypeMessage(respMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: fmt.Sprintf("/{path=--/%v}", strings.ToLower(r.Kind)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddApply adds a read method for the resource, along with
// any required messages.
func AddApply(r *parser.ParsedResource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Apply" + r.Kind + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Apply%v method", r.Kind),
	})
	addPathField(r, mb)
	addResourceField(r, resourceMb, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Apply"+r.Kind,
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Apply method for %v.", r.Plural),
	})
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Put{
			Put: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
		// TODO: do a conversion to underscores instead.
		Body: strings.ToLower(r.Kind),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func generateHTTPPath(r *parser.ParsedResource) string {
	elements := []string{strings.ToLower(r.Plural)}
	if len(r.Parents) > 0 {
		// TODO: handle multiple parents
		p := r.Parents[0]
		for p != nil {
			elements = append([]string{strings.ToLower(p.Plural)}, elements...)
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
		parentPath = generateHTTPPath(r.Parents[0])
		// parentPath = fmt.Sprintf("{parent=%v/}", generateHTTPPath(r.Parents[0]))
	}
	return fmt.Sprintf("/{parent=%v%v}", parentPath, strings.ToLower(r.Plural))
}

func addParentField(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	proto.SetExtension(o, annotations.E_ResourceReference, &annotations.ResourceReference{})
	f := builder.
		NewField(constants.FIELD_PARENT_NAME, builder.FieldTypeString()).
		SetNumber(constants.FIELD_PARENT_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("A field for the parent of %v", r.Kind),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addIdField(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_ID_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_ID_NUMBER).SetComments(builder.Comments{
		LeadingComment: "An id that uniquely identifies the resource within the collection",
	})
	mb.AddField(f)
}

func addPathField(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	proto.SetExtension(o, annotations.E_ResourceReference, &annotations.ResourceReference{
		Type: r.Type,
	})
	f := builder.NewField(constants.FIELD_PATH_NAME, builder.FieldTypeString()).
		SetNumber(constants.FIELD_PATH_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The globally unique identifier for the resource"),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addResourceField(r *parser.ParsedResource, resourceMb, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	f := builder.NewField(strings.ToLower(r.Kind), builder.FieldTypeMessage(resourceMb)).
		SetNumber(constants.FIELD_RESOURCE_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The resource to perform the operation on."),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addResourcesField(r *parser.ParsedResource, resourceMb, mb *builder.MessageBuilder) {
	f := builder.NewField("results", builder.FieldTypeMessage(resourceMb)).SetNumber(constants.FIELD_RESOURCES_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A list of %v", r.Plural),
	}).SetRepeated()
	mb.AddField(f)
}

func addPageToken(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_PAGE_TOKEN_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_PAGE_TOKEN_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("The page token indicating the starting point of the page"),
	})
	mb.AddField(f)
}

func addNextPageToken(r *parser.ParsedResource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_NEXT_PAGE_TOKEN_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_NEXT_PAGE_TOKEN_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("The page token indicating the ending point of this response."),
	})
	mb.AddField(f)
}
