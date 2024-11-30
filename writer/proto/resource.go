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
	"sort"
	"strings"

	"github.com/aep-dev/aep-lib-go/pkg/api"
	"github.com/aep-dev/aep-lib-go/pkg/openapi"
	"github.com/aep-dev/aepc/constants"
	"github.com/aep-dev/aepc/internal/utils"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// AddResource adds a resource's protos and RPCs to a file and service.
func AddResource(r *api.Resource, a *api.API, fb *builder.FileBuilder, sb *builder.ServiceBuilder, m *MessageStorage) error {
	// Do not recreate resources if they've already been created.
	resourceMb, ok := m.Messages[fmt.Sprintf("%s/%s", a.Name, r.Singular)]
	if !ok {
		return fmt.Errorf("%s not found in message storage", r.Singular)
	}

	if r.CreateMethod != nil {
		err := AddCreate(a, r, resourceMb, fb, sb)
		if err != nil {
			return err
		}
	}
	if r.GetMethod != nil {
		err := AddGet(a, r, resourceMb, fb, sb)
		if err != nil {
			return err
		}
	}
	if r.UpdateMethod != nil {
		err := AddUpdate(a, r, resourceMb, fb, sb)
		if err != nil {
			return err
		}
	}
	if r.DeleteMethod != nil {
		err := AddDelete(a, r, resourceMb, fb, sb)
		if err != nil {
			return err
		}
	}
	if r.ListMethod != nil {
		err := AddList(r, resourceMb, fb, sb)
		if err != nil {
			return err
		}
	}
	// TODO: add global list
	/// if r.GlobalList != nil {
	/// 	err := AddGlobalList(r, resourceMb, fb, sb)
	/// 	if err != nil {
	/// 		return err
	/// 	}
	/// }

	if r.ApplyMethod != nil {
		err := AddApply(a, r, resourceMb, fb, sb)
		if err != nil {
			return err
		}
	}
	return nil
}

// this function should only be called with openapi Schemas that
// map to primitive types.
func protoFieldType(name string, number int, s openapi.Schema, a *api.API, m *MessageStorage, parent *builder.MessageBuilder) (*builder.FieldType, error) {
	switch s.Type {
	case "object":
		typ, err := protoFieldTypeObject(name, &s, a, m, parent)
		if err != nil {
			return nil, fmt.Errorf("error creating proto type object for %s: %w", name, err)
		}
		return typ, nil
	// Ideally we would set the repeated parameter here as well.
	// But "repeated" is a property of the field, not the type.
	case "array":
		typ, err := protoFieldType(name, number, *s.Items, a, m, parent)
		if err != nil {
			return nil, fmt.Errorf("error creating proto type for array item for %s: %w", name, err)
		}
		return typ, nil
	case "string":
		return builder.FieldTypeString(), nil
	case "boolean":
		return builder.FieldTypeBool(), nil
	case "integer":
		if s.Format == "int32" {
			return builder.FieldTypeInt32(), nil
		} else if s.Format == "int64" {
			return builder.FieldTypeInt64(), nil
		}
	case "number":
		if s.Format == "float" {
			return builder.FieldTypeFloat(), nil
		} else if s.Format == "double" {
			return builder.FieldTypeDouble(), nil
		}
	}
	return nil, fmt.Errorf("proto type for %q, format %q not found", s.Type, s.Format)
}

func GenerateMessage(name string, s *openapi.Schema, a *api.API, m *MessageStorage) (*builder.MessageBuilder, error) {
	mb := builder.NewMessage(name)
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A %v.", name),
	})
	sorted_field_numbers := []int{}
	for n := range s.XAEPFieldNumbers {
		sorted_field_numbers = append(sorted_field_numbers, n)
	}
	sort.Ints(sorted_field_numbers)

	required := map[string]bool{}
	for _, n := range s.Required {
		required[n] = true
	}

	for _, num := range sorted_field_numbers {
		name := s.XAEPFieldNumbers[num]
		f, err := protoField(name, num, s.Properties[name], a, m, mb)
		if err != nil {
			return nil, err
		}
		if required[name] {
			o := &descriptorpb.FieldOptions{}
			proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
			f.SetOptions(o)
		}
		mb.AddField(f)
	}
	return mb, nil
}

func protoField(name string, number int, s openapi.Schema, a *api.API, m *MessageStorage, parent *builder.MessageBuilder) (*builder.FieldBuilder, error) {
	typ, err := protoFieldType(name, number, s, a, m, parent)
	if err != nil {
		return nil, fmt.Errorf("error creating proto field for %s: %w", name, err)
	}
	f := builder.NewField(name, typ).SetNumber(int32(number)).SetComments(
		builder.Comments{
			LeadingComment: fmt.Sprintf("Field for %v.", name),
		},
	)
	if s.Type == "array" {
		f.SetRepeated()
	}
	return f, nil
}

func protoFieldTypeObject(name string, s *openapi.Schema, a *api.API, m *MessageStorage, parent *builder.MessageBuilder) (*builder.FieldType, error) {
	if s.Ref != "" {
		wantedType := s.Ref
		// extract the name from the ref
		wantedType = strings.TrimPrefix(wantedType, "#/components/schemas/")
		wantedType = fmt.Sprintf("%s/%s", a.Name, wantedType)
		_, ok := m.Messages[wantedType]
		if !ok {
			return nil, fmt.Errorf("could not find message %s, referenced by %s", wantedType, name)
		}
		return builder.FieldTypeMessage(m.Messages[wantedType]), nil
	} else {
		msg, err := GenerateMessage(toMessageName(name), s, a, m)
		if err != nil {
			return nil, err
		}
		parent.AddNestedMessage(msg)
		return builder.FieldTypeMessage(msg), nil
	}
}

// GenerateResourceMesssage adds the resource message.
func GenerateSchemaMessage(name string, s *openapi.Schema, a *api.API, m *MessageStorage) (*builder.MessageBuilder, error) {
	mb, err := GenerateMessage(toMessageName(name), s, a, m)
	if err != nil {
		return nil, err
	}
	m.Messages[fmt.Sprintf("%s/%s", a.Name, name)] = mb
	return mb, nil
}

func AddCreate(a *api.API, r *api.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Create" + toMessageName(r.Singular) + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A Create request for a  %v resource.", r.Singular),
	})
	addParentField(r, mb)
	if r.CreateMethod.SupportsUserSettableCreate {
		addIdField(r, mb)
	}
	addResourceField(r, resourceMb, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Create"+toMessageName(r.Singular),
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Create method for %v.", r.Singular),
	})
	options := &descriptorpb.MethodOptions{}
	bodyField := utils.KebabToSnakeCase(r.Singular)
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Post{
			Post: generateParentHTTPPath(r),
		},
		Body: bodyField,
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{constants.FIELD_PARENT_NAME, bodyField}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddGet adds a read method for the resource, along with
// any required messages.
func AddGet(a *api.API, r *api.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Get" + toMessageName(r.Singular) + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Get%v method", r.Singular),
	})
	addPathField(a, r, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Get"+toMessageName(r.Singular),
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Get method for %v.", r.Singular),
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
func AddUpdate(a *api.API, r *api.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Update" + toMessageName(r.Singular) + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Update%v method", toMessageName(r.Singular)),
	})
	addPathField(a, r, mb)
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
	method := builder.NewMethod("Update"+toMessageName(r.Singular),
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeMessage(resourceMb, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Update method for %v.", r.Singular),
	})
	options := &descriptorpb.MethodOptions{}
	body_field := utils.KebabToSnakeCase(r.Singular)
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Patch{
			Patch: fmt.Sprintf("/{path=%v}", generateHTTPPath(r)),
		},
		Body: body_field,
	})
	proto.SetExtension(options, annotations.E_MethodSignature, []string{
		strings.Join([]string{body_field, constants.FIELD_UPDATE_MASK_NAME}, ","),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func AddDelete(a *api.API, r *api.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	mb := builder.NewMessage("Delete" + toMessageName(r.Singular) + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Delete%v method", toMessageName(r.Singular)),
	})
	addPathField(a, r, mb)
	fb.AddMessage(mb)
	emptyMd, err := desc.LoadMessageDescriptor("google.protobuf.Empty")
	if err != nil {
		return err
	}
	method := builder.NewMethod("Delete"+toMessageName(r.Singular),
		builder.RpcTypeMessage(mb, false),
		builder.RpcTypeImportedMessage(emptyMd, false),
	)
	method.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("An aep-compliant Delete method for %v.", r.Singular),
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

func AddList(r *api.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	reqMb := builder.NewMessage("List" + toMessageName(r.Plural) + "Request")
	reqMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the List%v method", r.Singular),
	})
	addParentField(r, reqMb)
	addPageToken(r, reqMb)
	reqMb.AddField(builder.NewField(constants.FIELD_MAX_PAGE_SIZE_NAME, builder.FieldTypeInt32()).
		SetNumber(constants.FIELD_MAX_PAGE_SIZE_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The maximum number of resources to return in a single page."),
		}))
	fb.AddMessage(reqMb)
	respMb := builder.NewMessage("List" + toMessageName(r.Plural) + "Response")
	respMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Response message for the List%v method", r.Singular),
	})
	addResourcesField(r, resourceMb, respMb)
	addNextPageToken(r, respMb)
	fb.AddMessage(respMb)
	method := builder.NewMethod("List"+toMessageName(r.Plural),
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

func AddGlobalList(r *api.Resource, a *api.API, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	// add the resource message
	// create request messages
	reqMb := builder.NewMessage("GlobalList" + r.Singular + "Request")
	reqMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the GlobalList%v method", r.Singular),
	})
	addPathField(a, r, reqMb)
	addPageToken(r, reqMb)
	fb.AddMessage(reqMb)
	respMb := builder.NewMessage("GlobalList" + r.Singular + "Response")
	respMb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Response message for the GlobalList%v method", r.Singular),
	})
	addResourcesField(r, resourceMb, respMb)
	addNextPageToken(r, respMb)
	fb.AddMessage(respMb)
	method := builder.NewMethod("GlobalList"+r.Singular,
		builder.RpcTypeMessage(reqMb, false),
		builder.RpcTypeMessage(respMb, false),
	)
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{
			Get: fmt.Sprintf("/{path=--/%v}", strings.ToLower(r.Singular)),
		},
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

// AddApply adds a read method for the resource, along with
// any required messages.
func AddApply(a *api.API, r *api.Resource, resourceMb *builder.MessageBuilder, fb *builder.FileBuilder, sb *builder.ServiceBuilder) error {
	mb := builder.NewMessage("Apply" + toMessageName(r.Singular) + "Request")
	mb.SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("Request message for the Apply%v method", r.Singular),
	})
	addPathField(a, r, mb)
	addResourceField(r, resourceMb, mb)
	fb.AddMessage(mb)
	method := builder.NewMethod("Apply"+toMessageName(r.Singular),
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
		Body: strings.ToLower(r.Singular),
	})
	method.SetOptions(options)
	sb.AddMethod(method)
	return nil
}

func generateHTTPPath(r *api.Resource) string {
	elements := []string{api.CollectionName(r)}
	if len(r.Parents) > 0 {
		// TODO: handle multiple parents
		p := r.Parents[0]
		for p != nil {
			elements = append([]string{api.CollectionName(p)}, elements...)
			if len(p.Parents) == 0 {
				break
			}
			p = p.Parents[0]
		}
	}
	return fmt.Sprintf("%v/*", strings.Join(elements, "/*/"))
}

func generateParentHTTPPath(r *api.Resource) string {
	parentPath := ""
	if len(r.Parents) == 0 {
		return fmt.Sprintf("/{parent=%v}", strings.ToLower(r.Plural))
	}
	if len(r.Parents) > 0 {
		parentPath = fmt.Sprintf("%v", generateHTTPPath(r.Parents[0]))
	}
	return fmt.Sprintf("/{parent=%v}/%v", parentPath, api.CollectionName(r))
}

func addParentField(r *api.Resource, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	proto.SetExtension(o, annotations.E_ResourceReference, &annotations.ResourceReference{})
	f := builder.
		NewField(constants.FIELD_PARENT_NAME, builder.FieldTypeString()).
		SetNumber(constants.FIELD_PARENT_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("A field for the parent of %v", r.Singular),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addIdField(r *api.Resource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_ID_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_ID_NUMBER).SetComments(builder.Comments{
		LeadingComment: "An id that uniquely identifies the resource within the collection",
	})
	mb.AddField(f)
}

func addPathField(a *api.API, r *api.Resource, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	proto.SetExtension(o, annotations.E_ResourceReference, &annotations.ResourceReference{
		Type: fmt.Sprintf("%v/%v", a.Name, r.Singular),
	})
	f := builder.NewField(constants.FIELD_PATH_NAME, builder.FieldTypeString()).
		SetNumber(constants.FIELD_PATH_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The globally unique identifier for the resource"),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addResourceField(r *api.Resource, resourceMb, mb *builder.MessageBuilder) {
	o := &descriptorpb.FieldOptions{}
	proto.SetExtension(o, annotations.E_FieldBehavior, []annotations.FieldBehavior{annotations.FieldBehavior_REQUIRED})
	f := builder.NewField(utils.KebabToSnakeCase(r.Singular), builder.FieldTypeMessage(resourceMb)).
		SetNumber(constants.FIELD_RESOURCE_NUMBER).
		SetComments(builder.Comments{
			LeadingComment: fmt.Sprintf("The resource to perform the operation on."),
		}).
		SetOptions(o)
	mb.AddField(f)
}

func addResourcesField(r *api.Resource, resourceMb, mb *builder.MessageBuilder) {
	f := builder.NewField("results", builder.FieldTypeMessage(resourceMb)).SetNumber(constants.FIELD_RESOURCES_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("A list of %v", r.Plural),
	}).SetRepeated()
	mb.AddField(f)
}

func addPageToken(r *api.Resource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_PAGE_TOKEN_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_PAGE_TOKEN_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("The page token indicating the starting point of the page"),
	})
	mb.AddField(f)
}

func addNextPageToken(r *api.Resource, mb *builder.MessageBuilder) {
	f := builder.NewField(constants.FIELD_NEXT_PAGE_TOKEN_NAME, builder.FieldTypeString()).SetNumber(constants.FIELD_NEXT_PAGE_TOKEN_NUMBER).SetComments(builder.Comments{
		LeadingComment: fmt.Sprintf("The page token indicating the ending point of this response."),
	})
	mb.AddField(f)
}

func getSortedProperties(s *openapi.Schema) []openapi.Schema {
	sorted_field_names := []string{}
	for _, f := range s.XAEPFieldNumbers {
		sorted_field_names = append(sorted_field_names, f)
	}
	sort.Strings(sorted_field_names)
	sorted_fields := []openapi.Schema{}
	for _, f := range sorted_field_names {
		sorted_fields = append(sorted_fields, s.Properties[f])
	}
	return sorted_fields
}
