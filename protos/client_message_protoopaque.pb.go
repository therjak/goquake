// SPDX-License-Identifier: GPL-2.0-or-later

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.0
// 	protoc        v5.29.2
// source: client_message.proto

//go:build protoopaque

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

type UsrCmd struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_MessageTime float32                `protobuf:"fixed32,1,opt,name=message_time,json=messageTime" json:"message_time,omitempty"`
	xxx_hidden_Pitch       float32                `protobuf:"fixed32,2,opt,name=pitch" json:"pitch,omitempty"`
	xxx_hidden_Yaw         float32                `protobuf:"fixed32,3,opt,name=yaw" json:"yaw,omitempty"`
	xxx_hidden_Roll        float32                `protobuf:"fixed32,4,opt,name=roll" json:"roll,omitempty"`
	xxx_hidden_Forward     float32                `protobuf:"fixed32,5,opt,name=forward" json:"forward,omitempty"`
	xxx_hidden_Side        float32                `protobuf:"fixed32,6,opt,name=side" json:"side,omitempty"`
	xxx_hidden_Up          float32                `protobuf:"fixed32,7,opt,name=up" json:"up,omitempty"`
	xxx_hidden_Attack      bool                   `protobuf:"varint,8,opt,name=attack" json:"attack,omitempty"`
	xxx_hidden_Jump        bool                   `protobuf:"varint,9,opt,name=jump" json:"jump,omitempty"`
	xxx_hidden_Impulse     int32                  `protobuf:"varint,10,opt,name=impulse" json:"impulse,omitempty"`
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *UsrCmd) Reset() {
	*x = UsrCmd{}
	mi := &file_client_message_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UsrCmd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UsrCmd) ProtoMessage() {}

func (x *UsrCmd) ProtoReflect() protoreflect.Message {
	mi := &file_client_message_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *UsrCmd) GetMessageTime() float32 {
	if x != nil {
		return x.xxx_hidden_MessageTime
	}
	return 0
}

func (x *UsrCmd) GetPitch() float32 {
	if x != nil {
		return x.xxx_hidden_Pitch
	}
	return 0
}

func (x *UsrCmd) GetYaw() float32 {
	if x != nil {
		return x.xxx_hidden_Yaw
	}
	return 0
}

func (x *UsrCmd) GetRoll() float32 {
	if x != nil {
		return x.xxx_hidden_Roll
	}
	return 0
}

func (x *UsrCmd) GetForward() float32 {
	if x != nil {
		return x.xxx_hidden_Forward
	}
	return 0
}

func (x *UsrCmd) GetSide() float32 {
	if x != nil {
		return x.xxx_hidden_Side
	}
	return 0
}

func (x *UsrCmd) GetUp() float32 {
	if x != nil {
		return x.xxx_hidden_Up
	}
	return 0
}

func (x *UsrCmd) GetAttack() bool {
	if x != nil {
		return x.xxx_hidden_Attack
	}
	return false
}

func (x *UsrCmd) GetJump() bool {
	if x != nil {
		return x.xxx_hidden_Jump
	}
	return false
}

func (x *UsrCmd) GetImpulse() int32 {
	if x != nil {
		return x.xxx_hidden_Impulse
	}
	return 0
}

func (x *UsrCmd) SetMessageTime(v float32) {
	x.xxx_hidden_MessageTime = v
}

func (x *UsrCmd) SetPitch(v float32) {
	x.xxx_hidden_Pitch = v
}

func (x *UsrCmd) SetYaw(v float32) {
	x.xxx_hidden_Yaw = v
}

func (x *UsrCmd) SetRoll(v float32) {
	x.xxx_hidden_Roll = v
}

func (x *UsrCmd) SetForward(v float32) {
	x.xxx_hidden_Forward = v
}

func (x *UsrCmd) SetSide(v float32) {
	x.xxx_hidden_Side = v
}

func (x *UsrCmd) SetUp(v float32) {
	x.xxx_hidden_Up = v
}

func (x *UsrCmd) SetAttack(v bool) {
	x.xxx_hidden_Attack = v
}

func (x *UsrCmd) SetJump(v bool) {
	x.xxx_hidden_Jump = v
}

func (x *UsrCmd) SetImpulse(v int32) {
	x.xxx_hidden_Impulse = v
}

type UsrCmd_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	MessageTime float32
	Pitch       float32
	Yaw         float32
	Roll        float32
	Forward     float32
	Side        float32
	Up          float32
	Attack      bool
	Jump        bool
	Impulse     int32
}

func (b0 UsrCmd_builder) Build() *UsrCmd {
	m0 := &UsrCmd{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_MessageTime = b.MessageTime
	x.xxx_hidden_Pitch = b.Pitch
	x.xxx_hidden_Yaw = b.Yaw
	x.xxx_hidden_Roll = b.Roll
	x.xxx_hidden_Forward = b.Forward
	x.xxx_hidden_Side = b.Side
	x.xxx_hidden_Up = b.Up
	x.xxx_hidden_Attack = b.Attack
	x.xxx_hidden_Jump = b.Jump
	x.xxx_hidden_Impulse = b.Impulse
	return m0
}

type Cmd struct {
	state            protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Union isCmd_Union            `protobuf_oneof:"union"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *Cmd) Reset() {
	*x = Cmd{}
	mi := &file_client_message_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Cmd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cmd) ProtoMessage() {}

func (x *Cmd) ProtoReflect() protoreflect.Message {
	mi := &file_client_message_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Cmd) GetDisconnect() bool {
	if x != nil {
		if x, ok := x.xxx_hidden_Union.(*cmd_Disconnect); ok {
			return x.Disconnect
		}
	}
	return false
}

func (x *Cmd) GetStringCmd() string {
	if x != nil {
		if x, ok := x.xxx_hidden_Union.(*cmd_StringCmd); ok {
			return x.StringCmd
		}
	}
	return ""
}

func (x *Cmd) GetMoveCmd() *UsrCmd {
	if x != nil {
		if x, ok := x.xxx_hidden_Union.(*cmd_MoveCmd); ok {
			return x.MoveCmd
		}
	}
	return nil
}

func (x *Cmd) SetDisconnect(v bool) {
	x.xxx_hidden_Union = &cmd_Disconnect{v}
}

func (x *Cmd) SetStringCmd(v string) {
	x.xxx_hidden_Union = &cmd_StringCmd{v}
}

func (x *Cmd) SetMoveCmd(v *UsrCmd) {
	if v == nil {
		x.xxx_hidden_Union = nil
		return
	}
	x.xxx_hidden_Union = &cmd_MoveCmd{v}
}

func (x *Cmd) HasUnion() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Union != nil
}

func (x *Cmd) HasDisconnect() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Union.(*cmd_Disconnect)
	return ok
}

func (x *Cmd) HasStringCmd() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Union.(*cmd_StringCmd)
	return ok
}

func (x *Cmd) HasMoveCmd() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Union.(*cmd_MoveCmd)
	return ok
}

func (x *Cmd) ClearUnion() {
	x.xxx_hidden_Union = nil
}

func (x *Cmd) ClearDisconnect() {
	if _, ok := x.xxx_hidden_Union.(*cmd_Disconnect); ok {
		x.xxx_hidden_Union = nil
	}
}

func (x *Cmd) ClearStringCmd() {
	if _, ok := x.xxx_hidden_Union.(*cmd_StringCmd); ok {
		x.xxx_hidden_Union = nil
	}
}

func (x *Cmd) ClearMoveCmd() {
	if _, ok := x.xxx_hidden_Union.(*cmd_MoveCmd); ok {
		x.xxx_hidden_Union = nil
	}
}

const Cmd_Union_not_set_case case_Cmd_Union = 0
const Cmd_Disconnect_case case_Cmd_Union = 1
const Cmd_StringCmd_case case_Cmd_Union = 2
const Cmd_MoveCmd_case case_Cmd_Union = 3

func (x *Cmd) WhichUnion() case_Cmd_Union {
	if x == nil {
		return Cmd_Union_not_set_case
	}
	switch x.xxx_hidden_Union.(type) {
	case *cmd_Disconnect:
		return Cmd_Disconnect_case
	case *cmd_StringCmd:
		return Cmd_StringCmd_case
	case *cmd_MoveCmd:
		return Cmd_MoveCmd_case
	default:
		return Cmd_Union_not_set_case
	}
}

type Cmd_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Fields of oneof xxx_hidden_Union:
	Disconnect *bool
	StringCmd  *string
	MoveCmd    *UsrCmd
	// -- end of xxx_hidden_Union
}

func (b0 Cmd_builder) Build() *Cmd {
	m0 := &Cmd{}
	b, x := &b0, m0
	_, _ = b, x
	if b.Disconnect != nil {
		x.xxx_hidden_Union = &cmd_Disconnect{*b.Disconnect}
	}
	if b.StringCmd != nil {
		x.xxx_hidden_Union = &cmd_StringCmd{*b.StringCmd}
	}
	if b.MoveCmd != nil {
		x.xxx_hidden_Union = &cmd_MoveCmd{b.MoveCmd}
	}
	return m0
}

type case_Cmd_Union protoreflect.FieldNumber

func (x case_Cmd_Union) String() string {
	md := file_client_message_proto_msgTypes[1].Descriptor()
	if x == 0 {
		return "not set"
	}
	return protoimpl.X.MessageFieldStringOf(md, protoreflect.FieldNumber(x))
}

type isCmd_Union interface {
	isCmd_Union()
}

type cmd_Disconnect struct {
	Disconnect bool `protobuf:"varint,1,opt,name=disconnect,oneof"`
}

type cmd_StringCmd struct {
	StringCmd string `protobuf:"bytes,2,opt,name=string_cmd,json=stringCmd,oneof"`
}

type cmd_MoveCmd struct {
	MoveCmd *UsrCmd `protobuf:"bytes,3,opt,name=move_cmd,json=moveCmd,oneof"`
}

func (*cmd_Disconnect) isCmd_Union() {}

func (*cmd_StringCmd) isCmd_Union() {}

func (*cmd_MoveCmd) isCmd_Union() {}

type ClientMessage struct {
	state           protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Cmds *[]*Cmd                `protobuf:"bytes,1,rep,name=cmds" json:"cmds,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *ClientMessage) Reset() {
	*x = ClientMessage{}
	mi := &file_client_message_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientMessage) ProtoMessage() {}

func (x *ClientMessage) ProtoReflect() protoreflect.Message {
	mi := &file_client_message_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ClientMessage) GetCmds() []*Cmd {
	if x != nil {
		if x.xxx_hidden_Cmds != nil {
			return *x.xxx_hidden_Cmds
		}
	}
	return nil
}

func (x *ClientMessage) SetCmds(v []*Cmd) {
	x.xxx_hidden_Cmds = &v
}

type ClientMessage_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Cmds []*Cmd
}

func (b0 ClientMessage_builder) Build() *ClientMessage {
	m0 := &ClientMessage{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Cmds = &b.Cmds
	return m0
}

var File_client_message_proto protoreflect.FileDescriptor

var file_client_message_proto_rawDesc = []byte{
	0x0a, 0x14, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x1a, 0x21,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x67, 0x6f, 0x5f, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xeb, 0x01, 0x0a, 0x06, 0x55, 0x73, 0x72, 0x43, 0x6d, 0x64, 0x12, 0x21, 0x0a, 0x0c,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x02, 0x52, 0x0b, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x14, 0x0a, 0x05, 0x70, 0x69, 0x74, 0x63, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x05,
	0x70, 0x69, 0x74, 0x63, 0x68, 0x12, 0x10, 0x0a, 0x03, 0x79, 0x61, 0x77, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x02, 0x52, 0x03, 0x79, 0x61, 0x77, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x6f, 0x6c, 0x6c, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x02, 0x52, 0x04, 0x72, 0x6f, 0x6c, 0x6c, 0x12, 0x18, 0x0a, 0x07, 0x66,
	0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x02, 0x52, 0x07, 0x66, 0x6f,
	0x72, 0x77, 0x61, 0x72, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x64, 0x65, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x02, 0x52, 0x04, 0x73, 0x69, 0x64, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x75, 0x70, 0x18,
	0x07, 0x20, 0x01, 0x28, 0x02, 0x52, 0x02, 0x75, 0x70, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x74, 0x74,
	0x61, 0x63, 0x6b, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x61, 0x74, 0x74, 0x61, 0x63,
	0x6b, 0x12, 0x12, 0x0a, 0x04, 0x6a, 0x75, 0x6d, 0x70, 0x18, 0x09, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x04, 0x6a, 0x75, 0x6d, 0x70, 0x12, 0x18, 0x0a, 0x07, 0x69, 0x6d, 0x70, 0x75, 0x6c, 0x73, 0x65,
	0x18, 0x0a, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x69, 0x6d, 0x70, 0x75, 0x6c, 0x73, 0x65, 0x22,
	0x7e, 0x0a, 0x03, 0x43, 0x6d, 0x64, 0x12, 0x20, 0x0a, 0x0a, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x6e,
	0x6e, 0x65, 0x63, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x48, 0x00, 0x52, 0x0a, 0x64, 0x69,
	0x73, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x12, 0x1f, 0x0a, 0x0a, 0x73, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x5f, 0x63, 0x6d, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x09,
	0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x43, 0x6d, 0x64, 0x12, 0x2b, 0x0a, 0x08, 0x6d, 0x6f, 0x76,
	0x65, 0x5f, 0x63, 0x6d, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x55, 0x73, 0x72, 0x43, 0x6d, 0x64, 0x48, 0x00, 0x52, 0x07, 0x6d,
	0x6f, 0x76, 0x65, 0x43, 0x6d, 0x64, 0x42, 0x07, 0x0a, 0x05, 0x75, 0x6e, 0x69, 0x6f, 0x6e, 0x22,
	0x30, 0x0a, 0x0d, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x12, 0x1f, 0x0a, 0x04, 0x63, 0x6d, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0b,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x43, 0x6d, 0x64, 0x52, 0x04, 0x63, 0x6d, 0x64,
	0x73, 0x42, 0x33, 0x5a, 0x21, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x74, 0x68, 0x65, 0x72, 0x6a, 0x61, 0x6b, 0x2f, 0x67, 0x6f, 0x71, 0x75, 0x61, 0x6b, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x92, 0x03, 0x0d, 0xd2, 0x3e, 0x02, 0x10, 0x02, 0x08, 0x02,
	0x10, 0x01, 0x20, 0x02, 0x30, 0x01, 0x62, 0x08, 0x65, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x70, 0xe8, 0x07,
}

var file_client_message_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_client_message_proto_goTypes = []any{
	(*UsrCmd)(nil),        // 0: protos.UsrCmd
	(*Cmd)(nil),           // 1: protos.Cmd
	(*ClientMessage)(nil), // 2: protos.ClientMessage
}
var file_client_message_proto_depIdxs = []int32{
	0, // 0: protos.Cmd.move_cmd:type_name -> protos.UsrCmd
	1, // 1: protos.ClientMessage.cmds:type_name -> protos.Cmd
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_client_message_proto_init() }
func file_client_message_proto_init() {
	if File_client_message_proto != nil {
		return
	}
	file_client_message_proto_msgTypes[1].OneofWrappers = []any{
		(*cmd_Disconnect)(nil),
		(*cmd_StringCmd)(nil),
		(*cmd_MoveCmd)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_client_message_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_client_message_proto_goTypes,
		DependencyIndexes: file_client_message_proto_depIdxs,
		MessageInfos:      file_client_message_proto_msgTypes,
	}.Build()
	File_client_message_proto = out.File
	file_client_message_proto_rawDesc = nil
	file_client_message_proto_goTypes = nil
	file_client_message_proto_depIdxs = nil
}
