// Code generated by protoc-gen-go. DO NOT EDIT.
// source: char.proto

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	char.proto
	common.proto
	ssproto.proto

It has these top-level messages:
	MessageCharLoginReq
	MessageCharLoginRes
	UserData
	BaseNoti
	MessageErrorNoti
	MessagePing
	MessageSyncServer
*/
package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type MessageCharLoginReq struct {
	AccountID int64  `protobuf:"varint,2,opt,name=AccountID" json:"AccountID,omitempty"`
	ClientVer string `protobuf:"bytes,3,opt,name=ClientVer" json:"ClientVer,omitempty"`
	UserToken string `protobuf:"bytes,4,opt,name=UserToken" json:"UserToken,omitempty"`
}

func (m *MessageCharLoginReq) Reset()                    { *m = MessageCharLoginReq{} }
func (m *MessageCharLoginReq) String() string            { return proto.CompactTextString(m) }
func (*MessageCharLoginReq) ProtoMessage()               {}
func (*MessageCharLoginReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *MessageCharLoginReq) GetAccountID() int64 {
	if m != nil {
		return m.AccountID
	}
	return 0
}

func (m *MessageCharLoginReq) GetClientVer() string {
	if m != nil {
		return m.ClientVer
	}
	return ""
}

func (m *MessageCharLoginReq) GetUserToken() string {
	if m != nil {
		return m.UserToken
	}
	return ""
}

type MessageCharLoginRes struct {
	AccountID int64     `protobuf:"varint,1,opt,name=AccountID" json:"AccountID,omitempty"`
	Base      *BaseNoti `protobuf:"bytes,2,opt,name=Base" json:"Base,omitempty"`
	Result    int32     `protobuf:"varint,3,opt,name=Result" json:"Result,omitempty"`
	IsNew     int32     `protobuf:"varint,5,opt,name=IsNew" json:"IsNew,omitempty"`
	ServerID  int32     `protobuf:"varint,6,opt,name=ServerID" json:"ServerID,omitempty"`
}

func (m *MessageCharLoginRes) Reset()                    { *m = MessageCharLoginRes{} }
func (m *MessageCharLoginRes) String() string            { return proto.CompactTextString(m) }
func (*MessageCharLoginRes) ProtoMessage()               {}
func (*MessageCharLoginRes) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *MessageCharLoginRes) GetAccountID() int64 {
	if m != nil {
		return m.AccountID
	}
	return 0
}

func (m *MessageCharLoginRes) GetBase() *BaseNoti {
	if m != nil {
		return m.Base
	}
	return nil
}

func (m *MessageCharLoginRes) GetResult() int32 {
	if m != nil {
		return m.Result
	}
	return 0
}

func (m *MessageCharLoginRes) GetIsNew() int32 {
	if m != nil {
		return m.IsNew
	}
	return 0
}

func (m *MessageCharLoginRes) GetServerID() int32 {
	if m != nil {
		return m.ServerID
	}
	return 0
}

func init() {
	proto.RegisterType((*MessageCharLoginReq)(nil), "pb.MessageCharLoginReq")
	proto.RegisterType((*MessageCharLoginRes)(nil), "pb.MessageCharLoginRes")
}

func init() { proto.RegisterFile("char.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 218 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0x4a, 0xce, 0x48, 0x2c,
	0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x48, 0x92, 0xe2, 0x49, 0xce, 0xcf, 0xcd,
	0xcd, 0xcf, 0x83, 0x88, 0x28, 0xe5, 0x73, 0x09, 0xfb, 0xa6, 0x16, 0x17, 0x27, 0xa6, 0xa7, 0x3a,
	0x03, 0x95, 0xf9, 0xe4, 0xa7, 0x67, 0xe6, 0x05, 0xa5, 0x16, 0x0a, 0xc9, 0x70, 0x71, 0x3a, 0x26,
	0x27, 0xe7, 0x97, 0xe6, 0x95, 0x78, 0xba, 0x48, 0x30, 0x29, 0x30, 0x6a, 0x30, 0x07, 0x21, 0x04,
	0x40, 0xb2, 0xce, 0x39, 0x99, 0xa9, 0x79, 0x25, 0x61, 0xa9, 0x45, 0x12, 0xcc, 0x40, 0x59, 0xce,
	0x20, 0x84, 0x00, 0x48, 0x36, 0xb4, 0x38, 0xb5, 0x28, 0x24, 0x3f, 0x3b, 0x35, 0x4f, 0x82, 0x05,
	0x22, 0x0b, 0x17, 0x50, 0x9a, 0xcf, 0x88, 0xcd, 0xc6, 0x62, 0x54, 0x1b, 0x19, 0xd1, 0x6d, 0x54,
	0xe0, 0x62, 0x71, 0x4a, 0x2c, 0x4e, 0x05, 0x3b, 0x85, 0xdb, 0x88, 0x47, 0xaf, 0x20, 0x49, 0x0f,
	0xc4, 0xf7, 0xcb, 0x2f, 0xc9, 0x0c, 0x02, 0xcb, 0x08, 0x89, 0x71, 0xb1, 0x01, 0x8d, 0x29, 0xcd,
	0x29, 0x01, 0x3b, 0x88, 0x35, 0x08, 0xca, 0x13, 0x12, 0xe1, 0x62, 0xf5, 0x2c, 0xf6, 0x4b, 0x2d,
	0x97, 0x60, 0x05, 0x0b, 0x43, 0x38, 0x42, 0x52, 0x5c, 0x1c, 0xc1, 0xa9, 0x45, 0x65, 0xa9, 0x45,
	0x40, 0xcb, 0xd8, 0xc0, 0x12, 0x70, 0x7e, 0x12, 0x1b, 0x38, 0x64, 0x8c, 0x01, 0x01, 0x00, 0x00,
	0xff, 0xff, 0x20, 0xa0, 0x16, 0xe1, 0x39, 0x01, 0x00, 0x00,
}