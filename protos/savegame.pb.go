// SPDX-License-Identifier: GPL-2.0-or-later

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.0
// 	protoc        v5.29.2
// source: savegame.proto

//go:build !protoopaque

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

type Vector struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	X             float32                `protobuf:"fixed32,1,opt,name=x" json:"x,omitempty"`
	Y             float32                `protobuf:"fixed32,2,opt,name=y" json:"y,omitempty"`
	Z             float32                `protobuf:"fixed32,3,opt,name=z" json:"z,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Vector) Reset() {
	*x = Vector{}
	mi := &file_savegame_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Vector) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Vector) ProtoMessage() {}

func (x *Vector) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Vector) GetX() float32 {
	if x != nil {
		return x.X
	}
	return 0
}

func (x *Vector) GetY() float32 {
	if x != nil {
		return x.Y
	}
	return 0
}

func (x *Vector) GetZ() float32 {
	if x != nil {
		return x.Z
	}
	return 0
}

func (x *Vector) SetX(v float32) {
	x.X = v
}

func (x *Vector) SetY(v float32) {
	x.Y = v
}

func (x *Vector) SetZ(v float32) {
	x.Z = v
}

type Vector_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	X float32
	Y float32
	Z float32
}

func (b0 Vector_builder) Build() *Vector {
	m0 := &Vector{}
	b, x := &b0, m0
	_, _ = b, x
	x.X = b.X
	x.Y = b.Y
	x.Z = b.Z
	return m0
}

type StringDef struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"` // sname
	Value         string                 `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StringDef) Reset() {
	*x = StringDef{}
	mi := &file_savegame_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StringDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StringDef) ProtoMessage() {}

func (x *StringDef) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *StringDef) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *StringDef) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *StringDef) SetId(v string) {
	x.Id = v
}

func (x *StringDef) SetValue(v string) {
	x.Value = v
}

type StringDef_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Id    string
	Value string
}

func (b0 StringDef_builder) Build() *StringDef {
	m0 := &StringDef{}
	b, x := &b0, m0
	_, _ = b, x
	x.Id = b.Id
	x.Value = b.Value
	return m0
}

type EntityDef struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"` // sname
	Value         int32                  `protobuf:"varint,2,opt,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EntityDef) Reset() {
	*x = EntityDef{}
	mi := &file_savegame_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EntityDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EntityDef) ProtoMessage() {}

func (x *EntityDef) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *EntityDef) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *EntityDef) GetValue() int32 {
	if x != nil {
		return x.Value
	}
	return 0
}

func (x *EntityDef) SetId(v string) {
	x.Id = v
}

func (x *EntityDef) SetValue(v int32) {
	x.Value = v
}

type EntityDef_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Id    string
	Value int32
}

func (b0 EntityDef_builder) Build() *EntityDef {
	m0 := &EntityDef{}
	b, x := &b0, m0
	_, _ = b, x
	x.Id = b.Id
	x.Value = b.Value
	return m0
}

type FunctionDef struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"` // sname
	Value         string                 `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FunctionDef) Reset() {
	*x = FunctionDef{}
	mi := &file_savegame_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FunctionDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FunctionDef) ProtoMessage() {}

func (x *FunctionDef) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *FunctionDef) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *FunctionDef) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *FunctionDef) SetId(v string) {
	x.Id = v
}

func (x *FunctionDef) SetValue(v string) {
	x.Value = v
}

type FunctionDef_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Id    string
	Value string
}

func (b0 FunctionDef_builder) Build() *FunctionDef {
	m0 := &FunctionDef{}
	b, x := &b0, m0
	_, _ = b, x
	x.Id = b.Id
	x.Value = b.Value
	return m0
}

type FieldDef struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"` // sname
	Value         string                 `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FieldDef) Reset() {
	*x = FieldDef{}
	mi := &file_savegame_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FieldDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FieldDef) ProtoMessage() {}

func (x *FieldDef) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *FieldDef) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *FieldDef) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *FieldDef) SetId(v string) {
	x.Id = v
}

func (x *FieldDef) SetValue(v string) {
	x.Value = v
}

type FieldDef_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Id    string
	Value string
}

func (b0 FieldDef_builder) Build() *FieldDef {
	m0 := &FieldDef{}
	b, x := &b0, m0
	_, _ = b, x
	x.Id = b.Id
	x.Value = b.Value
	return m0
}

type FloatDef struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"` // sname
	Value         float32                `protobuf:"fixed32,2,opt,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FloatDef) Reset() {
	*x = FloatDef{}
	mi := &file_savegame_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FloatDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FloatDef) ProtoMessage() {}

func (x *FloatDef) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *FloatDef) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *FloatDef) GetValue() float32 {
	if x != nil {
		return x.Value
	}
	return 0
}

func (x *FloatDef) SetId(v string) {
	x.Id = v
}

func (x *FloatDef) SetValue(v float32) {
	x.Value = v
}

type FloatDef_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Id    string
	Value float32
}

func (b0 FloatDef_builder) Build() *FloatDef {
	m0 := &FloatDef{}
	b, x := &b0, m0
	_, _ = b, x
	x.Id = b.Id
	x.Value = b.Value
	return m0
}

type VectorDef struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"` // sname
	Value         *Vector                `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *VectorDef) Reset() {
	*x = VectorDef{}
	mi := &file_savegame_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *VectorDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VectorDef) ProtoMessage() {}

func (x *VectorDef) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *VectorDef) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *VectorDef) GetValue() *Vector {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *VectorDef) SetId(v string) {
	x.Id = v
}

func (x *VectorDef) SetValue(v *Vector) {
	x.Value = v
}

func (x *VectorDef) HasValue() bool {
	if x == nil {
		return false
	}
	return x.Value != nil
}

func (x *VectorDef) ClearValue() {
	x.Value = nil
}

type VectorDef_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Id    string
	Value *Vector
}

func (b0 VectorDef_builder) Build() *VectorDef {
	m0 := &VectorDef{}
	b, x := &b0, m0
	_, _ = b, x
	x.Id = b.Id
	x.Value = b.Value
	return m0
}

type Globals struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// only globaldefs
	Entities      []*EntityDef `protobuf:"bytes,1,rep,name=entities" json:"entities,omitempty"`
	Floats        []*FloatDef  `protobuf:"bytes,2,rep,name=floats" json:"floats,omitempty"`
	Strings       []*StringDef `protobuf:"bytes,3,rep,name=strings" json:"strings,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Globals) Reset() {
	*x = Globals{}
	mi := &file_savegame_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Globals) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Globals) ProtoMessage() {}

func (x *Globals) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Globals) GetEntities() []*EntityDef {
	if x != nil {
		return x.Entities
	}
	return nil
}

func (x *Globals) GetFloats() []*FloatDef {
	if x != nil {
		return x.Floats
	}
	return nil
}

func (x *Globals) GetStrings() []*StringDef {
	if x != nil {
		return x.Strings
	}
	return nil
}

func (x *Globals) SetEntities(v []*EntityDef) {
	x.Entities = v
}

func (x *Globals) SetFloats(v []*FloatDef) {
	x.Floats = v
}

func (x *Globals) SetStrings(v []*StringDef) {
	x.Strings = v
}

type Globals_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// only globaldefs
	Entities []*EntityDef
	Floats   []*FloatDef
	Strings  []*StringDef
}

func (b0 Globals_builder) Build() *Globals {
	m0 := &Globals{}
	b, x := &b0, m0
	_, _ = b, x
	x.Entities = b.Entities
	x.Floats = b.Floats
	x.Strings = b.Strings
	return m0
}

type Edict struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// only fielddefs + alpha
	Entities      []*EntityDef   `protobuf:"bytes,1,rep,name=entities" json:"entities,omitempty"`
	Fields        []*FieldDef    `protobuf:"bytes,2,rep,name=fields" json:"fields,omitempty"`
	Floats        []*FloatDef    `protobuf:"bytes,3,rep,name=floats" json:"floats,omitempty"`
	Functions     []*FunctionDef `protobuf:"bytes,4,rep,name=functions" json:"functions,omitempty"`
	Strings       []*StringDef   `protobuf:"bytes,5,rep,name=strings" json:"strings,omitempty"`
	Vectors       []*VectorDef   `protobuf:"bytes,6,rep,name=vectors" json:"vectors,omitempty"`
	Alpha         float32        `protobuf:"fixed32,8,opt,name=alpha" json:"alpha,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Edict) Reset() {
	*x = Edict{}
	mi := &file_savegame_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Edict) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Edict) ProtoMessage() {}

func (x *Edict) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Edict) GetEntities() []*EntityDef {
	if x != nil {
		return x.Entities
	}
	return nil
}

func (x *Edict) GetFields() []*FieldDef {
	if x != nil {
		return x.Fields
	}
	return nil
}

func (x *Edict) GetFloats() []*FloatDef {
	if x != nil {
		return x.Floats
	}
	return nil
}

func (x *Edict) GetFunctions() []*FunctionDef {
	if x != nil {
		return x.Functions
	}
	return nil
}

func (x *Edict) GetStrings() []*StringDef {
	if x != nil {
		return x.Strings
	}
	return nil
}

func (x *Edict) GetVectors() []*VectorDef {
	if x != nil {
		return x.Vectors
	}
	return nil
}

func (x *Edict) GetAlpha() float32 {
	if x != nil {
		return x.Alpha
	}
	return 0
}

func (x *Edict) SetEntities(v []*EntityDef) {
	x.Entities = v
}

func (x *Edict) SetFields(v []*FieldDef) {
	x.Fields = v
}

func (x *Edict) SetFloats(v []*FloatDef) {
	x.Floats = v
}

func (x *Edict) SetFunctions(v []*FunctionDef) {
	x.Functions = v
}

func (x *Edict) SetStrings(v []*StringDef) {
	x.Strings = v
}

func (x *Edict) SetVectors(v []*VectorDef) {
	x.Vectors = v
}

func (x *Edict) SetAlpha(v float32) {
	x.Alpha = v
}

type Edict_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// only fielddefs + alpha
	Entities  []*EntityDef
	Fields    []*FieldDef
	Floats    []*FloatDef
	Functions []*FunctionDef
	Strings   []*StringDef
	Vectors   []*VectorDef
	Alpha     float32
}

func (b0 Edict_builder) Build() *Edict {
	m0 := &Edict{}
	b, x := &b0, m0
	_, _ = b, x
	x.Entities = b.Entities
	x.Fields = b.Fields
	x.Floats = b.Floats
	x.Functions = b.Functions
	x.Strings = b.Strings
	x.Vectors = b.Vectors
	x.Alpha = b.Alpha
	return m0
}

type SaveGame struct {
	state         protoimpl.MessageState `protogen:"hybrid.v1"`
	Comment       string                 `protobuf:"bytes,1,opt,name=comment" json:"comment,omitempty"`
	SpawnParams   []float32              `protobuf:"fixed32,2,rep,packed,name=spawn_params,json=spawnParams" json:"spawn_params,omitempty"`
	CurrentSkill  int32                  `protobuf:"varint,3,opt,name=current_skill,json=currentSkill" json:"current_skill,omitempty"`
	MapName       string                 `protobuf:"bytes,4,opt,name=map_name,json=mapName" json:"map_name,omitempty"`
	MapTime       float32                `protobuf:"fixed32,5,opt,name=map_time,json=mapTime" json:"map_time,omitempty"`
	LightStyles   []string               `protobuf:"bytes,6,rep,name=light_styles,json=lightStyles" json:"light_styles,omitempty"`
	Globals       *Globals               `protobuf:"bytes,7,opt,name=globals" json:"globals,omitempty"`
	Edicts        []*Edict               `protobuf:"bytes,8,rep,name=edicts" json:"edicts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SaveGame) Reset() {
	*x = SaveGame{}
	mi := &file_savegame_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SaveGame) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaveGame) ProtoMessage() {}

func (x *SaveGame) ProtoReflect() protoreflect.Message {
	mi := &file_savegame_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *SaveGame) GetComment() string {
	if x != nil {
		return x.Comment
	}
	return ""
}

func (x *SaveGame) GetSpawnParams() []float32 {
	if x != nil {
		return x.SpawnParams
	}
	return nil
}

func (x *SaveGame) GetCurrentSkill() int32 {
	if x != nil {
		return x.CurrentSkill
	}
	return 0
}

func (x *SaveGame) GetMapName() string {
	if x != nil {
		return x.MapName
	}
	return ""
}

func (x *SaveGame) GetMapTime() float32 {
	if x != nil {
		return x.MapTime
	}
	return 0
}

func (x *SaveGame) GetLightStyles() []string {
	if x != nil {
		return x.LightStyles
	}
	return nil
}

func (x *SaveGame) GetGlobals() *Globals {
	if x != nil {
		return x.Globals
	}
	return nil
}

func (x *SaveGame) GetEdicts() []*Edict {
	if x != nil {
		return x.Edicts
	}
	return nil
}

func (x *SaveGame) SetComment(v string) {
	x.Comment = v
}

func (x *SaveGame) SetSpawnParams(v []float32) {
	x.SpawnParams = v
}

func (x *SaveGame) SetCurrentSkill(v int32) {
	x.CurrentSkill = v
}

func (x *SaveGame) SetMapName(v string) {
	x.MapName = v
}

func (x *SaveGame) SetMapTime(v float32) {
	x.MapTime = v
}

func (x *SaveGame) SetLightStyles(v []string) {
	x.LightStyles = v
}

func (x *SaveGame) SetGlobals(v *Globals) {
	x.Globals = v
}

func (x *SaveGame) SetEdicts(v []*Edict) {
	x.Edicts = v
}

func (x *SaveGame) HasGlobals() bool {
	if x == nil {
		return false
	}
	return x.Globals != nil
}

func (x *SaveGame) ClearGlobals() {
	x.Globals = nil
}

type SaveGame_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Comment      string
	SpawnParams  []float32
	CurrentSkill int32
	MapName      string
	MapTime      float32
	LightStyles  []string
	Globals      *Globals
	Edicts       []*Edict
}

func (b0 SaveGame_builder) Build() *SaveGame {
	m0 := &SaveGame{}
	b, x := &b0, m0
	_, _ = b, x
	x.Comment = b.Comment
	x.SpawnParams = b.SpawnParams
	x.CurrentSkill = b.CurrentSkill
	x.MapName = b.MapName
	x.MapTime = b.MapTime
	x.LightStyles = b.LightStyles
	x.Globals = b.Globals
	x.Edicts = b.Edicts
	return m0
}

var File_savegame_proto protoreflect.FileDescriptor

var file_savegame_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x73, 0x61, 0x76, 0x65, 0x67, 0x61, 0x6d, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x1a, 0x21, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x67, 0x6f, 0x5f, 0x66, 0x65, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x32, 0x0a, 0x06, 0x56,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x0c, 0x0a, 0x01, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x01, 0x78, 0x12, 0x0c, 0x0a, 0x01, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01,
	0x79, 0x12, 0x0c, 0x0a, 0x01, 0x7a, 0x18, 0x03, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x7a, 0x22,
	0x31, 0x0a, 0x09, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x44, 0x65, 0x66, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x22, 0x31, 0x0a, 0x09, 0x45, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x44, 0x65, 0x66, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x33, 0x0a, 0x0b, 0x46, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x44, 0x65, 0x66, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x30, 0x0a, 0x08, 0x46, 0x69,
	0x65, 0x6c, 0x64, 0x44, 0x65, 0x66, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x30, 0x0a, 0x08,
	0x46, 0x6c, 0x6f, 0x61, 0x74, 0x44, 0x65, 0x66, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x41,
	0x0a, 0x09, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x44, 0x65, 0x66, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x24, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x73, 0x2e, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x22, 0x8f, 0x01, 0x0a, 0x07, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x73, 0x12, 0x2d, 0x0a,
	0x08, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x45, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x44,
	0x65, 0x66, 0x52, 0x08, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x69, 0x65, 0x73, 0x12, 0x28, 0x0a, 0x06,
	0x66, 0x6c, 0x6f, 0x61, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x46, 0x6c, 0x6f, 0x61, 0x74, 0x44, 0x65, 0x66, 0x52, 0x06,
	0x66, 0x6c, 0x6f, 0x61, 0x74, 0x73, 0x12, 0x2b, 0x0a, 0x07, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
	0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x44, 0x65, 0x66, 0x52, 0x07, 0x73, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x73, 0x22, 0xad, 0x02, 0x0a, 0x05, 0x45, 0x64, 0x69, 0x63, 0x74, 0x12, 0x2d, 0x0a,
	0x08, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x45, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x44,
	0x65, 0x66, 0x52, 0x08, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x69, 0x65, 0x73, 0x12, 0x28, 0x0a, 0x06,
	0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x44, 0x65, 0x66, 0x52, 0x06,
	0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x12, 0x28, 0x0a, 0x06, 0x66, 0x6c, 0x6f, 0x61, 0x74, 0x73,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e,
	0x46, 0x6c, 0x6f, 0x61, 0x74, 0x44, 0x65, 0x66, 0x52, 0x06, 0x66, 0x6c, 0x6f, 0x61, 0x74, 0x73,
	0x12, 0x31, 0x0a, 0x09, 0x66, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x04, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x46, 0x75, 0x6e,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x65, 0x66, 0x52, 0x09, 0x66, 0x75, 0x6e, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x12, 0x2b, 0x0a, 0x07, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x73, 0x18, 0x05,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x44, 0x65, 0x66, 0x52, 0x07, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x73,
	0x12, 0x2b, 0x0a, 0x07, 0x76, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x56, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x44, 0x65, 0x66, 0x52, 0x07, 0x76, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x12, 0x14, 0x0a,
	0x05, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x18, 0x08, 0x20, 0x01, 0x28, 0x02, 0x52, 0x05, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x22, 0x97, 0x02, 0x0a, 0x08, 0x53, 0x61, 0x76, 0x65, 0x47, 0x61, 0x6d, 0x65,
	0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x73, 0x70,
	0x61, 0x77, 0x6e, 0x5f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x02,
	0x52, 0x0b, 0x73, 0x70, 0x61, 0x77, 0x6e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x12, 0x23, 0x0a,
	0x0d, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x6b, 0x69, 0x6c, 0x6c, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x53, 0x6b, 0x69,
	0x6c, 0x6c, 0x12, 0x19, 0x0a, 0x08, 0x6d, 0x61, 0x70, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x61, 0x70, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x19, 0x0a,
	0x08, 0x6d, 0x61, 0x70, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x02, 0x52,
	0x07, 0x6d, 0x61, 0x70, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x6c, 0x69, 0x67, 0x68,
	0x74, 0x5f, 0x73, 0x74, 0x79, 0x6c, 0x65, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0b,
	0x6c, 0x69, 0x67, 0x68, 0x74, 0x53, 0x74, 0x79, 0x6c, 0x65, 0x73, 0x12, 0x29, 0x0a, 0x07, 0x67,
	0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x73, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x73, 0x52, 0x07, 0x67,
	0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x73, 0x12, 0x25, 0x0a, 0x06, 0x65, 0x64, 0x69, 0x63, 0x74, 0x73,
	0x18, 0x08, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e,
	0x45, 0x64, 0x69, 0x63, 0x74, 0x52, 0x06, 0x65, 0x64, 0x69, 0x63, 0x74, 0x73, 0x42, 0x33, 0x5a,
	0x21, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x68, 0x65, 0x72,
	0x6a, 0x61, 0x6b, 0x2f, 0x67, 0x6f, 0x71, 0x75, 0x61, 0x6b, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x92, 0x03, 0x0d, 0xd2, 0x3e, 0x02, 0x10, 0x02, 0x08, 0x02, 0x10, 0x01, 0x20, 0x02,
	0x30, 0x01, 0x62, 0x08, 0x65, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x70, 0xe8, 0x07,
}

var file_savegame_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_savegame_proto_goTypes = []any{
	(*Vector)(nil),      // 0: protos.Vector
	(*StringDef)(nil),   // 1: protos.StringDef
	(*EntityDef)(nil),   // 2: protos.EntityDef
	(*FunctionDef)(nil), // 3: protos.FunctionDef
	(*FieldDef)(nil),    // 4: protos.FieldDef
	(*FloatDef)(nil),    // 5: protos.FloatDef
	(*VectorDef)(nil),   // 6: protos.VectorDef
	(*Globals)(nil),     // 7: protos.Globals
	(*Edict)(nil),       // 8: protos.Edict
	(*SaveGame)(nil),    // 9: protos.SaveGame
}
var file_savegame_proto_depIdxs = []int32{
	0,  // 0: protos.VectorDef.value:type_name -> protos.Vector
	2,  // 1: protos.Globals.entities:type_name -> protos.EntityDef
	5,  // 2: protos.Globals.floats:type_name -> protos.FloatDef
	1,  // 3: protos.Globals.strings:type_name -> protos.StringDef
	2,  // 4: protos.Edict.entities:type_name -> protos.EntityDef
	4,  // 5: protos.Edict.fields:type_name -> protos.FieldDef
	5,  // 6: protos.Edict.floats:type_name -> protos.FloatDef
	3,  // 7: protos.Edict.functions:type_name -> protos.FunctionDef
	1,  // 8: protos.Edict.strings:type_name -> protos.StringDef
	6,  // 9: protos.Edict.vectors:type_name -> protos.VectorDef
	7,  // 10: protos.SaveGame.globals:type_name -> protos.Globals
	8,  // 11: protos.SaveGame.edicts:type_name -> protos.Edict
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_savegame_proto_init() }
func file_savegame_proto_init() {
	if File_savegame_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_savegame_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_savegame_proto_goTypes,
		DependencyIndexes: file_savegame_proto_depIdxs,
		MessageInfos:      file_savegame_proto_msgTypes,
	}.Build()
	File_savegame_proto = out.File
	file_savegame_proto_rawDesc = nil
	file_savegame_proto_goTypes = nil
	file_savegame_proto_depIdxs = nil
}
