package main

import (
	"crypto/cipher"
	"github.com/golang/protobuf/proto"
	"github.com/raochq/gogame/base/logger"
	"github.com/raochq/gogame/base/network"
	"github.com/raochq/gogame/errcode"
	"github.com/raochq/gogame/protocol"
	"github.com/raochq/gogame/protocol/pb"
	"time"
)

const (
	SESS_KEYEXCG       = 0x1  // 是否已经交换完毕KEY
	SESS_ENCRYPT       = 0x2  // 是否可以开始加密
	SESS_KICKED_OUT    = 0x4  // 踢掉
	SESS_AUTHORIZED    = 0x8  // 已授权访问
	SESS_REPLACED      = 0x10 // 登录替换
	SESS_GAMESVR_CLOSE = 0x20 // 服务器关闭
)

type ClientManager struct {
	clients map[int64]*ClientTask
}
type ClientTask struct {
	conn           *network.TCPConnection
	Encoder        cipher.Stream // 加密器
	Decoder        cipher.Stream // 解密器
	AccountID      int64         // 玩家ID(portal时已经从login中获得accountID)
	GSID           uint16        // 游戏服ID;(redis也有存)
	TSID           uint16        // 组队服务器ID
	Flag           int32         // 会话标记
	LastPacketTime int64         // 前一个包到达时间
}

func NewClientManager() *ClientManager {
	m := ClientManager{
		clients: make(map[int64]*ClientTask),
	}
	return &m
}
func (this *ClientManager) Connect(con *network.TCPConnection) {
	client := &ClientTask{
		conn:           con,
		LastPacketTime: time.Now().Unix(),
	}
	con.Owner = client
}
func (this *ClientManager) Disconnect(*network.TCPConnection) {}
func (this *ClientManager) Receive(con *network.TCPConnection, data []byte) {
	c := con.Owner.(ClientTask)
	// 解密
	if c.Flag&SESS_ENCRYPT != 0 {
		c.Decoder.XORKeyStream(data, data)
	}
	csMsg := &protocol.CSMessage{}
	err := csMsg.UnMarshal(data)
	if err != nil {
		logger.Error("Protocol unmarshal msg error(%v)", err)
		c.SendErrToMe(int32(errcode.EC_ProtobufUnmarshalFail), 0)
		c.TerminateClient(SESS_KICKED_OUT)
		return
	}
	msgID := csMsg.Head.MessageID
	packetNo := csMsg.Head.PacketNo
	msgBody := csMsg.Body
	bContinue, msgBody, ret := c.HookClientMessage(msgID, packetNo, msgBody)
	if ret != nil {
		logger.Error("HookClientMessage(%v) error:%v", msgID, ret)
		err := ret.(errcode.ErrCode)
		c.SendErrToMe(int32(err), packetNo)
		c.TerminateClient(SESS_KICKED_OUT)
		return
	}
	if bContinue == true {
		//todo 根据协议，发送到不同服务
	}
}
func (this *ClientManager) Init()  {}
func (this *ClientManager) Close() {}

func (c *ClientTask) SendMessageToMe(msg proto.Message) errcode.ErrCode {
	bMsgBody, msgID, err := protocol.Marshal(msg)
	if err != nil {
		logger.Error("SendMessageToMe %d protobuf marshal fail %s", c.AccountID, err.Error())
		return errcode.EC_ProtobufMarshalFail
	}
	csMsg := &protocol.CSMessage{}
	csMsg.Head.MessageID = msgID
	csMsg.Body = bMsgBody
	data, _ := csMsg.Marshal()
	if c.Flag&SESS_ENCRYPT != 0 {
		c.Encoder.XORKeyStream(data, data)
	}
	if ok := c.conn.Send(data); !ok {
		return errcode.EC_NetworkFail
	}
	return 0
}
func (c *ClientTask) TerminateClient(flag int32) {
	c.Flag |= flag
	time.Sleep(300 * time.Millisecond)
	c.Close()
}
func (c *ClientTask) SendErrToMe(errCode int32, packetNo uint32) {
	noti := &pb.MessageErrorNoti{
		ErrCode: errCode,
		Base: &pb.BaseNoti{
			PacketNo: packetNo,
			NowTime:  time.Now().Unix(),
		},
	}
	c.SendMessageToMe(noti)
}

func (c *ClientTask) Close() {
	c.conn.Shutdown()
}
func (c *ClientTask) HookClientMessage(msgID uint16, packetNo uint32, bMsgBody []byte) (bool, []byte, error) {
	var ret error
	switch msgID {
	case protocol.MsgID_MessageCharLoginReq:
		bMsgBody, ret = c.handlerLoginMsg(bMsgBody)
		if ret != nil {
			return false, bMsgBody, ret
		}
	case protocol.MsgID_MessagePing:
		return false, bMsgBody, nil
	default:
		logger.Debug("%d recived message %d", c.AccountID, msgID)
	}

	if !protocol.IsProtocolExist(msgID) {
		logger.Warn("Client send unknown msg(0x%x)", msgID)
		return false, bMsgBody, errcode.EC_UnknownMessage
	}
	return true, bMsgBody, nil
}

func (c *ClientTask) handlerLoginMsg(bMsgBody []byte) ([]byte, error) {
	logReq := &pb.MessageCharLoginReq{}
	err := proto.Unmarshal(bMsgBody, logReq)
	if err != nil {
		logger.Error("protobuf unmarsh(%v) error(%v)", bMsgBody, err)
		return bMsgBody, errcode.EC_ProtobufUnmarshalFail
	}
	if logReq.GetAccountID() <= 0 {
		logger.Error("handleLoginMsg fail as accountid(%v) invalid", logReq.GetAccountID())
		return bMsgBody, errcode.EC_ParameterInvalid
	}
	c.AccountID = logReq.GetAccountID()
	logger.Debug("handleLoginMsg player %d login server from addr %v", c.AccountID, c.conn.RemoteAddr())
	//todo 写登录逻辑
	return bMsgBody, nil
}
