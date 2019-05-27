// Code generated by protoc-gen-go. DO NOT EDIT.
// source: streaming.proto

package chatbothubstreaming

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type MessageTunnelRequest struct {
	Seq                  uint32   `protobuf:"varint,1,opt,name=seq,proto3" json:"seq,omitempty"`
	NeedAck              bool     `protobuf:"varint,2,opt,name=needAck,proto3" json:"needAck,omitempty"`
	EventName            string   `protobuf:"bytes,3,opt,name=eventName,proto3" json:"eventName,omitempty"`
	Payload              string   `protobuf:"bytes,4,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MessageTunnelRequest) Reset()         { *m = MessageTunnelRequest{} }
func (m *MessageTunnelRequest) String() string { return proto.CompactTextString(m) }
func (*MessageTunnelRequest) ProtoMessage()    {}
func (*MessageTunnelRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f2e8ceba11142904, []int{0}
}

func (m *MessageTunnelRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MessageTunnelRequest.Unmarshal(m, b)
}
func (m *MessageTunnelRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MessageTunnelRequest.Marshal(b, m, deterministic)
}
func (m *MessageTunnelRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MessageTunnelRequest.Merge(m, src)
}
func (m *MessageTunnelRequest) XXX_Size() int {
	return xxx_messageInfo_MessageTunnelRequest.Size(m)
}
func (m *MessageTunnelRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_MessageTunnelRequest.DiscardUnknown(m)
}

var xxx_messageInfo_MessageTunnelRequest proto.InternalMessageInfo

func (m *MessageTunnelRequest) GetSeq() uint32 {
	if m != nil {
		return m.Seq
	}
	return 0
}

func (m *MessageTunnelRequest) GetNeedAck() bool {
	if m != nil {
		return m.NeedAck
	}
	return false
}

func (m *MessageTunnelRequest) GetEventName() string {
	if m != nil {
		return m.EventName
	}
	return ""
}

func (m *MessageTunnelRequest) GetPayload() string {
	if m != nil {
		return m.Payload
	}
	return ""
}

type MessageTunnelResponseError struct {
	Code                 int64    `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Message              string   `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MessageTunnelResponseError) Reset()         { *m = MessageTunnelResponseError{} }
func (m *MessageTunnelResponseError) String() string { return proto.CompactTextString(m) }
func (*MessageTunnelResponseError) ProtoMessage()    {}
func (*MessageTunnelResponseError) Descriptor() ([]byte, []int) {
	return fileDescriptor_f2e8ceba11142904, []int{1}
}

func (m *MessageTunnelResponseError) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MessageTunnelResponseError.Unmarshal(m, b)
}
func (m *MessageTunnelResponseError) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MessageTunnelResponseError.Marshal(b, m, deterministic)
}
func (m *MessageTunnelResponseError) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MessageTunnelResponseError.Merge(m, src)
}
func (m *MessageTunnelResponseError) XXX_Size() int {
	return xxx_messageInfo_MessageTunnelResponseError.Size(m)
}
func (m *MessageTunnelResponseError) XXX_DiscardUnknown() {
	xxx_messageInfo_MessageTunnelResponseError.DiscardUnknown(m)
}

var xxx_messageInfo_MessageTunnelResponseError proto.InternalMessageInfo

func (m *MessageTunnelResponseError) GetCode() int64 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *MessageTunnelResponseError) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

type MessageTunnelResponse struct {
	Ack                  uint32                      `protobuf:"varint,1,opt,name=ack,proto3" json:"ack,omitempty"`
	Payload              string                      `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	Error                *MessageTunnelResponseError `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_unrecognized     []byte                      `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *MessageTunnelResponse) Reset()         { *m = MessageTunnelResponse{} }
func (m *MessageTunnelResponse) String() string { return proto.CompactTextString(m) }
func (*MessageTunnelResponse) ProtoMessage()    {}
func (*MessageTunnelResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f2e8ceba11142904, []int{2}
}

func (m *MessageTunnelResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MessageTunnelResponse.Unmarshal(m, b)
}
func (m *MessageTunnelResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MessageTunnelResponse.Marshal(b, m, deterministic)
}
func (m *MessageTunnelResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MessageTunnelResponse.Merge(m, src)
}
func (m *MessageTunnelResponse) XXX_Size() int {
	return xxx_messageInfo_MessageTunnelResponse.Size(m)
}
func (m *MessageTunnelResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MessageTunnelResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MessageTunnelResponse proto.InternalMessageInfo

func (m *MessageTunnelResponse) GetAck() uint32 {
	if m != nil {
		return m.Ack
	}
	return 0
}

func (m *MessageTunnelResponse) GetPayload() string {
	if m != nil {
		return m.Payload
	}
	return ""
}

func (m *MessageTunnelResponse) GetError() *MessageTunnelResponseError {
	if m != nil {
		return m.Error
	}
	return nil
}

func init() {
	proto.RegisterType((*MessageTunnelRequest)(nil), "chatbothubstreaming.MessageTunnelRequest")
	proto.RegisterType((*MessageTunnelResponseError)(nil), "chatbothubstreaming.MessageTunnelResponseError")
	proto.RegisterType((*MessageTunnelResponse)(nil), "chatbothubstreaming.MessageTunnelResponse")
}

func init() { proto.RegisterFile("streaming.proto", fileDescriptor_f2e8ceba11142904) }

var fileDescriptor_f2e8ceba11142904 = []byte{
	// 312 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0xdf, 0x4a, 0xc3, 0x30,
	0x14, 0xc6, 0xcd, 0x36, 0xff, 0xec, 0xc8, 0x50, 0x32, 0x85, 0x30, 0xbc, 0x18, 0x05, 0xa1, 0x7a,
	0x51, 0x65, 0x3e, 0x81, 0x1b, 0x03, 0x11, 0x94, 0x11, 0x7d, 0x81, 0xb4, 0x3d, 0xac, 0xa3, 0x6b,
	0xb2, 0x35, 0xa9, 0x45, 0x6f, 0xbd, 0xf3, 0xa9, 0x25, 0x19, 0x9d, 0x55, 0x2a, 0xec, 0xee, 0x9c,
	0x1e, 0xbe, 0xef, 0xfc, 0xbe, 0xe6, 0xc0, 0x89, 0x36, 0x39, 0x8a, 0x6c, 0x21, 0xe7, 0xc1, 0x2a,
	0x57, 0x46, 0xd1, 0x7e, 0x94, 0x08, 0x13, 0x2a, 0x93, 0x14, 0xe1, 0x76, 0xe4, 0x7d, 0xc0, 0xd9,
	0x13, 0x6a, 0x2d, 0xe6, 0xf8, 0x5a, 0x48, 0x89, 0x4b, 0x8e, 0xeb, 0x02, 0xb5, 0xa1, 0xa7, 0xd0,
	0xd6, 0xb8, 0x66, 0x64, 0x48, 0xfc, 0x1e, 0xb7, 0x25, 0x65, 0x70, 0x28, 0x11, 0xe3, 0xfb, 0x28,
	0x65, 0xad, 0x21, 0xf1, 0x8f, 0x78, 0xd5, 0xd2, 0x0b, 0xe8, 0xe2, 0x1b, 0x4a, 0xf3, 0x2c, 0x32,
	0x64, 0xed, 0x21, 0xf1, 0xbb, 0xfc, 0xe7, 0x83, 0xd5, 0xad, 0xc4, 0xfb, 0x52, 0x89, 0x98, 0x75,
	0xdc, 0xac, 0x6a, 0xbd, 0x47, 0x18, 0xfc, 0xd9, 0xad, 0x57, 0x4a, 0x6a, 0x9c, 0xe6, 0xb9, 0xca,
	0x29, 0x85, 0x4e, 0xa4, 0x62, 0x74, 0x08, 0x6d, 0xee, 0x6a, 0xeb, 0x95, 0x6d, 0x14, 0x8e, 0xa1,
	0xcb, 0xab, 0xd6, 0xfb, 0x22, 0x70, 0xde, 0x68, 0x66, 0x93, 0x88, 0x28, 0xad, 0x92, 0x88, 0x28,
	0xad, 0x13, 0xb5, 0x7e, 0x11, 0xd1, 0x29, 0xec, 0xa3, 0x5d, 0xee, 0x52, 0x1c, 0x8f, 0x6e, 0x82,
	0x86, 0x5f, 0x16, 0xfc, 0xcf, 0xcc, 0x37, 0xea, 0xd1, 0x27, 0x81, 0xfe, 0x24, 0x11, 0x66, 0xac,
	0xcc, 0x43, 0x11, 0xbe, 0x54, 0x4a, 0xba, 0x84, 0x5e, 0x56, 0x17, 0xd3, 0xab, 0x5d, 0x16, 0xb8,
	0x07, 0x19, 0x5c, 0xef, 0xce, 0xe2, 0xed, 0xf9, 0xe4, 0x96, 0x8c, 0x27, 0x70, 0x29, 0xd1, 0x04,
	0x89, 0x28, 0xd3, 0x72, 0x61, 0x92, 0x72, 0x21, 0xe3, 0x9a, 0x47, 0xb0, 0x35, 0x19, 0xb3, 0x06,
	0xd6, 0x99, 0x3d, 0x99, 0x19, 0x09, 0x0f, 0xdc, 0xed, 0xdc, 0x7d, 0x07, 0x00, 0x00, 0xff, 0xff,
	0x46, 0x02, 0x93, 0xc4, 0x4e, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ChatBotHubStreamingClient is the client API for ChatBotHubStreaming service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ChatBotHubStreamingClient interface {
	// 双向 tunnel
	MessageTunnel(ctx context.Context, opts ...grpc.CallOption) (ChatBotHubStreaming_MessageTunnelClient, error)
}

type chatBotHubStreamingClient struct {
	cc *grpc.ClientConn
}

func NewChatBotHubStreamingClient(cc *grpc.ClientConn) ChatBotHubStreamingClient {
	return &chatBotHubStreamingClient{cc}
}

func (c *chatBotHubStreamingClient) MessageTunnel(ctx context.Context, opts ...grpc.CallOption) (ChatBotHubStreaming_MessageTunnelClient, error) {
	stream, err := c.cc.NewStream(ctx, &_ChatBotHubStreaming_serviceDesc.Streams[0], "/chatbothubstreaming.ChatBotHubStreaming/messageTunnel", opts...)
	if err != nil {
		return nil, err
	}
	x := &chatBotHubStreamingMessageTunnelClient{stream}
	return x, nil
}

type ChatBotHubStreaming_MessageTunnelClient interface {
	Send(*MessageTunnelRequest) error
	Recv() (*MessageTunnelResponse, error)
	grpc.ClientStream
}

type chatBotHubStreamingMessageTunnelClient struct {
	grpc.ClientStream
}

func (x *chatBotHubStreamingMessageTunnelClient) Send(m *MessageTunnelRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *chatBotHubStreamingMessageTunnelClient) Recv() (*MessageTunnelResponse, error) {
	m := new(MessageTunnelResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ChatBotHubStreamingServer is the server API for ChatBotHubStreaming service.
type ChatBotHubStreamingServer interface {
	// 双向 tunnel
	MessageTunnel(ChatBotHubStreaming_MessageTunnelServer) error
}

// UnimplementedChatBotHubStreamingServer can be embedded to have forward compatible implementations.
type UnimplementedChatBotHubStreamingServer struct {
}

func (*UnimplementedChatBotHubStreamingServer) MessageTunnel(srv ChatBotHubStreaming_MessageTunnelServer) error {
	return status.Errorf(codes.Unimplemented, "method MessageTunnel not implemented")
}

func RegisterChatBotHubStreamingServer(s *grpc.Server, srv ChatBotHubStreamingServer) {
	s.RegisterService(&_ChatBotHubStreaming_serviceDesc, srv)
}

func _ChatBotHubStreaming_MessageTunnel_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ChatBotHubStreamingServer).MessageTunnel(&chatBotHubStreamingMessageTunnelServer{stream})
}

type ChatBotHubStreaming_MessageTunnelServer interface {
	Send(*MessageTunnelResponse) error
	Recv() (*MessageTunnelRequest, error)
	grpc.ServerStream
}

type chatBotHubStreamingMessageTunnelServer struct {
	grpc.ServerStream
}

func (x *chatBotHubStreamingMessageTunnelServer) Send(m *MessageTunnelResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *chatBotHubStreamingMessageTunnelServer) Recv() (*MessageTunnelRequest, error) {
	m := new(MessageTunnelRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _ChatBotHubStreaming_serviceDesc = grpc.ServiceDesc{
	ServiceName: "chatbothubstreaming.ChatBotHubStreaming",
	HandlerType: (*ChatBotHubStreamingServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "messageTunnel",
			Handler:       _ChatBotHubStreaming_MessageTunnel_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "streaming.proto",
}
