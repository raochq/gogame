package protocol

import (
	"encoding/binary"
	"errors"
	"math"
)

const (
	PACKET_LIMIT = 65535
	MsgHeadLen   = 8
)

type Packet struct {
	pos  int
	data []byte
}

func (p *Packet) Data() []byte {
	return p.data
}

func (p *Packet) Length() int {
	return len(p.data)
}

//=============================================== Readers
func (p *Packet) ReadBool() (ret bool, err error) {
	b, _err := p.ReadByte()

	if b != byte(1) {
		return false, _err
	}

	return true, _err
}

func (p *Packet) ReadByte() (ret byte, err error) {
	if p.pos >= len(p.data) {
		err = errors.New("read byte failed")
		return
	}

	ret = p.data[p.pos]
	p.pos++
	return
}

func (p *Packet) ReadBytes() (ret []byte, err error) {
	if p.pos+2 > len(p.data) {
		err = errors.New("read bytes header failed")
		return
	}
	size, _ := p.ReadU16()
	if p.pos+int(size) > len(p.data) {
		err = errors.New("read bytes data failed")
		return
	}

	ret = p.data[p.pos : p.pos+int(size)]
	p.pos += int(size)
	return
}

func (p *Packet) ReadRestBytes() (ret []byte, err error) {
	if len(p.data) <= 0 {
		err = errors.New("read rest bytes data failed")
		return
	}

	ret = p.data[p.pos:len(p.data)]
	p.pos += len(p.data)
	return
}

func (p *Packet) ReadString() (ret string, err error) {
	if p.pos+2 > len(p.data) {
		err = errors.New("read string header failed")
		return
	}

	size, _ := p.ReadU16()
	if p.pos+int(size) > len(p.data) {
		err = errors.New("read string data failed")
		return
	}

	bytes := p.data[p.pos : p.pos+int(size)]
	p.pos += int(size)
	ret = string(bytes)
	return
}

func (p *Packet) ReadS8() (ret int8, err error) {
	_ret, _err := p.ReadByte()
	ret = int8(_ret)
	err = _err
	return
}

func (p *Packet) ReadU16() (ret uint16, err error) {
	if p.pos+2 > len(p.data) {
		err = errors.New("read uint16 failed")
		return
	}

	buf := p.data[p.pos : p.pos+2]
	ret = uint16(buf[0])<<8 | uint16(buf[1])
	p.pos += 2
	return
}

func (p *Packet) ReadS16() (ret int16, err error) {
	_ret, _err := p.ReadU16()
	ret = int16(_ret)
	err = _err
	return
}

func (p *Packet) ReadU24() (ret uint32, err error) {
	if p.pos+3 > len(p.data) {
		err = errors.New("read uint24 failed")
		return
	}

	buf := p.data[p.pos : p.pos+3]
	ret = uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
	p.pos += 3
	return
}

func (p *Packet) ReadS24() (ret int32, err error) {
	_ret, _err := p.ReadU24()
	ret = int32(_ret)
	err = _err
	return
}

func (p *Packet) ReadU32() (ret uint32, err error) {
	if p.pos+4 > len(p.data) {
		err = errors.New("read uint32 failed")
		return
	}

	buf := p.data[p.pos : p.pos+4]
	ret = uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	p.pos += 4
	return
}

func (p *Packet) ReadS32() (ret int32, err error) {
	_ret, _err := p.ReadU32()
	ret = int32(_ret)
	err = _err
	return
}

func (p *Packet) ReadU64() (ret uint64, err error) {
	if p.pos+8 > len(p.data) {
		err = errors.New("read uint64 failed")
		return
	}

	ret = 0
	buf := p.data[p.pos : p.pos+8]
	for i, v := range buf {
		ret |= uint64(v) << uint((7-i)*8)
	}
	p.pos += 8
	return
}

func (p *Packet) ReadS64() (ret int64, err error) {
	_ret, _err := p.ReadU64()
	ret = int64(_ret)
	err = _err
	return
}

func (p *Packet) ReadFloat32() (ret float32, err error) {
	bits, _err := p.ReadU32()
	if _err != nil {
		return float32(0), _err
	}

	ret = math.Float32frombits(bits)
	if math.IsNaN(float64(ret)) || math.IsInf(float64(ret), 0) {
		return 0, nil
	}

	return ret, nil
}

func (p *Packet) ReadFloat64() (ret float64, err error) {
	bits, _err := p.ReadU64()
	if _err != nil {
		return float64(0), _err
	}

	ret = math.Float64frombits(bits)
	if math.IsNaN(ret) || math.IsInf(ret, 0) {
		return 0, nil
	}

	return ret, nil
}

//================================================ Writers
func (p *Packet) WriteZeros(n int) {
	for i := 0; i < n; i++ {
		p.data = append(p.data, byte(0))
	}
}

func (p *Packet) WriteBool(v bool) {
	if v {
		p.data = append(p.data, byte(1))
	} else {
		p.data = append(p.data, byte(0))
	}
}

func (p *Packet) WritesByte(v byte) {
	p.data = append(p.data, v)
}

func (p *Packet) WriteBytes(v []byte) {
	p.WriteU16(uint16(len(v)))
	p.data = append(p.data, v...)
}

func (p *Packet) WriteRawBytes(v []byte) {
	p.data = append(p.data, v...)
}

func (p *Packet) WriteString(v string) {
	bytes := []byte(v)
	p.WriteU16(uint16(len(bytes)))
	p.data = append(p.data, bytes...)
}

func (p *Packet) WriteS8(v int8) {
	p.WritesByte(byte(v))
}

func (p *Packet) WriteU16(v uint16) {
	p.data = append(p.data, byte(v>>8), byte(v))
}

func (p *Packet) WriteS16(v int16) {
	p.WriteU16(uint16(v))
}

func (p *Packet) WriteU24(v uint32) {
	p.data = append(p.data, byte(v>>16), byte(v>>8), byte(v))
}

func (p *Packet) WriteU32(v uint32) {
	p.data = append(p.data, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func (p *Packet) WriteS32(v int32) {
	p.WriteU32(uint32(v))
}

func (p *Packet) WriteU64(v uint64) {
	p.data = append(p.data, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func (p *Packet) WriteS64(v int64) {
	p.WriteU64(uint64(v))
}

func (p *Packet) WriteFloat32(f float32) {
	v := math.Float32bits(f)
	p.WriteU32(v)
}

func (p *Packet) WriteFloat64(f float64) {
	v := math.Float64bits(f)
	p.WriteU64(v)
}

func NewPacket(data []byte) *Packet {
	return &Packet{data: data}
}

var (
	errRecvLess = errors.New("Recv less")
	errRecvMore = errors.New("Recv more")
	errSendLess = errors.New("Send less")
)

// CSMessage include CSMessageHead & Body that transfer between client & server
// Body will UnMarshal according to MessageID
type CSMessageHead struct {
	MessageID uint16
	PacketNo  uint32
	Version   uint16
}

type CSMessage struct {
	Head CSMessageHead
	Body []byte
}

func (csMsg *CSMessage) Marshal() ([]byte, error) {
	buffer := make([]byte, 0, MsgHeadLen+len(csMsg.Body))
	writer := NewPacket(make([]byte, 0, PACKET_LIMIT))
	writer.WriteU16(csMsg.Head.MessageID)
	writer.WriteU32(csMsg.Head.PacketNo)
	writer.WriteU16(csMsg.Head.Version)
	writer.WriteRawBytes(csMsg.Body)
	return buffer, nil
}

func (csMsg *CSMessage) UnMarshal(data []byte) error {
	var err error = nil
	reader := NewPacket(data)
	csMsg.Head.MessageID, err = reader.ReadU16()
	if err != nil {
		return err
	}
	csMsg.Head.PacketNo, err = reader.ReadU32()
	if err != nil {
		return err
	}
	csMsg.Head.Version, err = reader.ReadU16()
	if err != nil {
		return err
	}
	csMsg.Body, err = reader.ReadRestBytes()
	if err != nil {
		return err
	}
	return nil
}

type SSMessageHead struct {
	SSCmd     int8
	TransType int8
	SrcType   int8
	DstType   int8
	SrcID     uint16
	DstID     uint16
}

type SSMessage struct {
	Head *SSMessageHead // SSMessageHead
	Body []byte         // SSMessageBody
}

type SSMessageBody struct {
	MessageID    uint16
	PacketNo     uint32
	SrcAccountID int64
	DstAccountID int64
	GSID         uint16
	Body         []byte
}

func (ssMsg *SSMessageBody) Marshal() ([]byte, error) {
	writer := NewPacket(make([]byte, 0, PACKET_LIMIT))
	writer.WriteU16(ssMsg.MessageID)
	writer.WriteU32(ssMsg.PacketNo)
	writer.WriteS64(ssMsg.SrcAccountID)
	writer.WriteS64(ssMsg.DstAccountID)
	writer.WriteU16(ssMsg.GSID)
	writer.WriteRawBytes(ssMsg.Body)
	return writer.Data(), nil
}

func (ssMsg *SSMessageBody) UnMarshal(data []byte) error {
	var err error = nil
	reader := NewPacket(data)
	ssMsg.MessageID, err = reader.ReadU16()
	if err != nil {
		return err
	}
	ssMsg.PacketNo, err = reader.ReadU32()
	if err != nil {
		return err
	}
	ssMsg.SrcAccountID, err = reader.ReadS64()
	if err != nil {
		return err
	}
	ssMsg.DstAccountID, err = reader.ReadS64()
	if err != nil {
		return err
	}
	ssMsg.GSID, err = reader.ReadU16()
	if err != nil {
		return err
	}
	ssMsg.Body, err = reader.ReadRestBytes()
	if err != nil {
		return err
	}
	return nil
}

func UnpackSSMessage(buff []byte) *SSMessage {
	if len(buff) < MsgHeadLen {
		return nil
	}
	ssMsg := new(SSMessage)
	head := new(SSMessageHead)
	head.SSCmd = int8(buff[0])
	head.TransType = int8(buff[1])
	head.SrcType = int8(buff[2])
	head.DstType = int8(buff[3])
	head.SrcID = binary.BigEndian.Uint16(buff[4:6])
	head.DstID = binary.BigEndian.Uint16(buff[6:8])
	ssMsg.Head = head
	ssMsg.Body = buff[8:]
	return ssMsg
}

func PackSSMessage(ssMsg *SSMessage) []byte {
	buf := make([]byte, MsgHeadLen, MsgHeadLen+len(ssMsg.Body))
	buf[0] = byte(ssMsg.Head.SSCmd)
	buf[1] = byte(ssMsg.Head.TransType)
	buf[2] = byte(ssMsg.Head.SrcType)
	buf[3] = byte(ssMsg.Head.DstType)
	binary.BigEndian.PutUint16(buf[4:], uint16(ssMsg.Head.SrcID))
	binary.BigEndian.PutUint16(buf[6:], uint16(ssMsg.Head.DstID))
	buf = append(buf, ssMsg.Body...)
	return buf
}
