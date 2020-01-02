package protocol

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"reflect"
)

//go:generate go run cmd/main.go
//go:generate go fmt protoFactory.go
const (
	Message_ID_MASK       = 0xff00
	Message_SSProto_Start = 0x0000
	Message_Gen_Start     = 0x0100
	Message_Char_Start    = 0x0200
	Message_IM_Start      = 0x4000
	Message_Team_Start    = 0x5000
	Message_SNS_Start     = 0x6000
)

type Processor struct {
	msgType map[uint16]reflect.Type
	msgID   map[reflect.Type]uint16
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.msgType = make(map[uint16]reflect.Type)
	p.msgID = make(map[reflect.Type]uint16)
	return p
}

// 注册协议。与marshal，Unmarshal同时使用时不安全。建议只在程序启动时使用
func (p *Processor) Register(id uint16, msg proto.Message) error {
	if _, ok := p.msgType[id]; ok {
		return fmt.Errorf("proto: duplicate proto type registered: %v", id)

	}
	t := reflect.TypeOf(msg)

	if _, ok := p.msgID[t]; ok {
		return fmt.Errorf("proto: duplicate proto type registered: %v", t.String())
	}
	p.msgType[id] = t
	p.msgID[t] = id
	return nil
}
func (p *Processor) RegisterType(id uint16, t reflect.Type) error {
	if _, ok := p.msgType[id]; ok {
		return fmt.Errorf("proto: duplicate proto type registered: %v", id)

	}

	if _, ok := p.msgID[t]; ok {
		return fmt.Errorf("proto: duplicate proto type registered: %v", t.String())
	}
	p.msgType[id] = t
	p.msgID[t] = id
	return nil
}

func (p *Processor) Unmarshal(id uint16, data []byte) (interface{}, error) {
	// msg
	if t, ok := p.msgType[id]; ok {
		msg := reflect.New(t.Elem()).Interface()
		return msg, proto.UnmarshalMerge(data[2:], msg.(proto.Message))
	} else {
		return nil, fmt.Errorf("protocol %d not registered", id)
	}
}

// 不可与Register同时执行
func (p *Processor) Marshal(msg interface{}) ([]byte, uint16, error) {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		err := fmt.Errorf("protocol %s not registered", msgType)
		return nil, 0, err
	}
	// data
	data, err := proto.Marshal(msg.(proto.Message))
	return data, id, err
}

func (p *Processor) IsProtocolExist(msgId uint16) bool {
	if _, ok := p.msgType[msgId]; ok {
		return true
	}
	return false
}

var processor *Processor

func Unmarshal(id uint16, data []byte) (interface{}, error) {
	return processor.Unmarshal(id, data)
}

func Marshal(msg interface{}) ([]byte, uint16, error) {
	return processor.Marshal(msg)
}

func IsProtocolExist(msgId uint16) bool {
	return processor.IsProtocolExist(msgId)
}
