// SPDX-License-Identifier: GPL-2.0-or-later

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.0
// 	protoc        v5.29.2
// source: history.proto

package protos

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/gofeaturespb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type History struct {
	state              protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Entries []string               `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *History) Reset() {
	*x = History{}
	mi := &file_history_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *History) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*History) ProtoMessage() {}

func (x *History) ProtoReflect() protoreflect.Message {
	mi := &file_history_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *History) GetEntries() []string {
	if x != nil {
		return x.xxx_hidden_Entries
	}
	return nil
}

func (x *History) SetEntries(v []string) {
	x.xxx_hidden_Entries = v
}

type History_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Entries []string
}

func (b0 History_builder) Build() *History {
	m0 := &History{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Entries = b.Entries
	return m0
}

var File_history_proto protoreflect.FileDescriptor

var file_history_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x68, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x1a, 0x21, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x67, 0x6f, 0x5f, 0x66, 0x65, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x23, 0x0a, 0x07, 0x48, 0x69,
	0x73, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x42,
	0x33, 0x5a, 0x21, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x68,
	0x65, 0x72, 0x6a, 0x61, 0x6b, 0x2f, 0x67, 0x6f, 0x71, 0x75, 0x61, 0x6b, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x92, 0x03, 0x0d, 0xd2, 0x3e, 0x02, 0x10, 0x03, 0x08, 0x02, 0x10, 0x01,
	0x20, 0x02, 0x30, 0x01, 0x62, 0x08, 0x65, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x70, 0xe8,
	0x07,
}

var file_history_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_history_proto_goTypes = []any{
	(*History)(nil), // 0: protos.History
}
var file_history_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_history_proto_init() }
func file_history_proto_init() {
	if File_history_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_history_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_history_proto_goTypes,
		DependencyIndexes: file_history_proto_depIdxs,
		MessageInfos:      file_history_proto_msgTypes,
	}.Build()
	File_history_proto = out.File
	file_history_proto_rawDesc = nil
	file_history_proto_goTypes = nil
	file_history_proto_depIdxs = nil
}
