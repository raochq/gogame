package main

import (
	"crypto/cipher"
	"github.com/golang/protobuf/proto"
	"gogame/base/conf"
	"gogame/base/logger"
	"gogame/base/network"
	. "gogame/common"
	. "gogame/errcode"
	"gogame/protocol"
	"gogame/protocol/pb"
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
		c.SendErrToMe(EC_UnmarshalFail, 0)
		c.TerminateClient(SESS_KICKED_OUT)
		return
	}
	msgID := csMsg.Head.MessageID
	packetNo := csMsg.Head.PacketNo
	msgBody := csMsg.Body
	bContinue, msgBody, ret := c.HookClientMessage(msgID, packetNo, msgBody)
	if ret != nil {
		logger.Error("HookClientMessage(%v) error:%v", msgID, ret)
		c.SendErrToMe(ret, packetNo)
		c.TerminateClient(SESS_KICKED_OUT)
		return
	}
	if bContinue == true {
		destType := GetMessageDestByMsgID(msgID)
		switch destType {
		case EntityType_GameSvr:
			ret := c.gamesvr_forward(msgID, packetNo, msgBody)
			if ret != nil {
				logger.Error("service id:%v execute failed, error:%v", msgID, ret)
				c.SendErrToMe(ret, packetNo)
				c.TerminateClient(SESS_KICKED_OUT)
			}
		case EntityType_IM:
			ret := c.imserver_forward(msgID, packetNo, msgBody)
			if ret != nil {
				logger.Error("send message to im fail %d", ret)
			}
		case EntityType_TeamSvr:
			ret := c.teamserver_forward(msgID, packetNo, msgBody)
			if ret != nil {
				logger.Error("message id:0x%x send to teamserver execute failed, error:%v", msgID, ret)
			}
		default:
			logger.Error("service id:%d has no destType:%v to precedue", msgID, destType)
			c.SendErrToMe(EC_UnknownMessage, packetNo)
			c.TerminateClient(SESS_KICKED_OUT)
		}
	}
}
func (this *ClientManager) Init()  {}
func (this *ClientManager) Close() {}

//发送消息到客户端
func (c *ClientTask) SendMessageToMe(msg proto.Message) error {
	bMsgBody, msgID, err := protocol.Marshal(msg)
	if err != nil {
		logger.Error("SendMessageToMe %d protobuf marshal fail %s", c.AccountID, err.Error())
		return EC_MarshalFail
	}
	csMsg := &protocol.CSMessage{}
	csMsg.Head.MessageID = msgID
	csMsg.Body = bMsgBody
	data, _ := csMsg.Marshal()
	if c.Flag&SESS_ENCRYPT != 0 {
		c.Encoder.XORKeyStream(data, data)
	}
	if ok := c.conn.Send(data); !ok {
		return EC_NetworkFail
	}
	return nil
}

//踢人
func (c *ClientTask) TerminateClient(flag int32) {
	c.Flag |= flag
	time.Sleep(300 * time.Millisecond)
	c.Close()
}

//发送错误码给客户端
func (c *ClientTask) SendErrToMe(err error, packetNo uint32) {
	code, ok := err.(ErrCode)
	if !ok {
		code = EC_Unknown
	}
	noti := &pb.MessageErrorNoti{
		ErrCode: int32(code),
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

// 处理收到的客户端消息
func (c *ClientTask) HookClientMessage(msgID uint16, packetNo uint32, bMsgBody []byte) (bool, []byte, error) {
	var ret error
	switch msgID {
	case protocol.MsgID_MessageCharLoginReq:
		bMsgBody, ret = c.handlerLoginMsg(bMsgBody)
		if ret != nil {
			return false, bMsgBody, ret
		}
	case protocol.MsgID_MessagePing:
		c.handlePing(packetNo)
		return false, bMsgBody, nil
	default:
		logger.Debug("%d recived message %d", c.AccountID, msgID)
	}

	if !protocol.IsProtocolExist(msgID) {
		logger.Warn("Client send unknown msg(0x%x)", msgID)
		return false, bMsgBody, EC_UnknownMessage
	}
	return true, bMsgBody, nil
}

//处理登录消息
func (c *ClientTask) handlerLoginMsg(bMsgBody []byte) ([]byte, error) {
	logReq := &pb.MessageCharLoginReq{}
	err := proto.Unmarshal(bMsgBody, logReq)
	if err != nil {
		logger.Error("protobuf unmarsh(%v) error(%v)", bMsgBody, err)
		return bMsgBody, EC_UnmarshalFail
	}
	if logReq.GetAccountID() <= 0 {
		logger.Error("handleLoginMsg fail as accountid(%v) invalid", logReq.GetAccountID())
		return bMsgBody, EC_ParameterInvalid
	}
	c.AccountID = logReq.GetAccountID()
	logger.Debug("handleLoginMsg player %d login server from addr %v", c.AccountID, c.conn.RemoteAddr())
	//todo 写登录逻辑
	return bMsgBody, nil
}

// 处理ping消息
func (c *ClientTask) handlePing(packetNo uint32) error {
	pong := &pb.MessagePong{
		Base: &pb.BaseNoti{
			PacketNo: packetNo,
			NowTime:  time.Now().Unix(),
		},
	}
	return c.SendMessageToMe(pong)
}
func (c *ClientTask) gamesvr_forward(msgID uint16, packetNo uint32, bMsgBody []byte) error {
	if c.GSID == INVALID_GAMESVR_ID {
		logger.Error("Gamesvr invalid(%v) fail, pls relogin", c.GSID)
		return EC_GamesvrError
	}

	ssMsg := &protocol.SSMessageBody{
		MessageID:    msgID,
		PacketNo:     packetNo,
		SrcAccountID: c.AccountID,
		DstAccountID: c.AccountID,
		GSID:         c.GSID,
		Body:         bMsgBody,
	}
	return SendMessageToRouter(EntityType_GameSvr, c.GSID, ssMsg)
}

// forward messages to team server
func (c *ClientTask) teamserver_forward(msgID uint16, packetNo uint32, bMsgBody []byte) error {
	ssMsg := &protocol.SSMessageBody{
		MessageID:    msgID,
		PacketNo:     packetNo,
		SrcAccountID: c.AccountID,
		DstAccountID: c.AccountID,
		GSID:         c.GSID,
		Body:         bMsgBody,
	}
	return SendMessageToRouter(EntityType_TeamSvr, c.TSID, ssMsg)
}

// forward messages to im server
func (c *ClientTask) imserver_forward(msgID uint16, packetNo uint32, bMsgBody []byte) error {
	ssMsg := &protocol.SSMessageBody{
		MessageID:    msgID,
		PacketNo:     packetNo,
		SrcAccountID: c.AccountID,
		DstAccountID: c.AccountID,
		GSID:         c.GSID,
		Body:         bMsgBody,
	}
	return SendMessageToIMSvr(ssMsg)
}

// 获取协议目的地址
// 根据协议范围段区分
func GetMessageDestByMsgID(msgID uint16) int32 {
	switch msgID & protocol.Message_ID_MASK {
	case protocol.Message_IM_Start:
		return EntityType_IM
	case protocol.Message_Team_Start:
		return EntityType_TeamSvr
	default:
		return EntityType_GameSvr
	}
}

func SendMessageToRouter(dstType int8, dstID uint16, ssMsg *protocol.SSMessageBody) error {
	head := &protocol.SSMessageHead{
		SSCmd:     SSCMD_MSG,
		TransType: TransType_ByKey,
		SrcType:   EntityType_Portal,
		DstType:   dstType,
		SrcID:     uint16(conf.Cfg.GameGate.ServerID),
		DstID:     uint16(dstID),
	}
	body, err := ssMsg.Marshal()
	if err != nil {
		logger.Error("SendMessageToRouter dstType %d dstID %d fail", dstType, dstID)
		return EC_MarshalFail
	}
	msg := &protocol.SSMessage{Head: head, Body: body}
	logger.Debug("SendMessageToRouter%v", msg)
	//var router *Router
	//if dstType == EntityType_TeamSvr {
	//	if dstID == 0 {
	//		svrId := RandTeamsvrId()
	//		router = GetCrossRouterBySvrID(svrId)
	//		head.DstID = svrId
	//	} else {
	//		router = GetCrossRouterBySvrID(dstID)
	//	}
	//} else {
	//	router = GetRouterBySvrID(dstID)
	//}
	//if router != nil {
	//	router.client.Write(msg.FormSendData())
	//	return nil
	//}
	//if dstType == EntityType_GameSvr && GameSvrAlive(dstID) == false {
	//	return RC_ServerClose
	//}
	//logger.Error("SendMessageToRouter dst %d router is nil", dstID)
	//return RC_RouterNotFound
	return nil
}

func SendMessageToIMSvr(ssMsg *protocol.SSMessageBody) error {
	//im := GetIMByAccountID(ssMsg.DstAccountID)
	//if im == nil {
	//	logger.Debug("SendMessageToIMSvr get im nil %d", ssMsg.DstAccountID)
	//	return EC_ParameterInvalid
	//}
	head := &protocol.SSMessageHead{
		SSCmd:     SSCMD_MSG,
		TransType: TransType_ByKey,
		SrcType:   EntityType_Portal,
		DstType:   EntityType_IM,
		SrcID:     uint16(conf.Cfg.GameGate.ServerID),
		//DstID:     im.id,
	}
	body, err := ssMsg.Marshal()
	if err != nil {
		//logger.Error("SendMessageToIMSvr dstID %d fail", im.id)
		return EC_MarshalFail
	}
	logger.Debug("SendMessageToIMSvr %d msg 0x%x", ssMsg.DstAccountID, ssMsg.MessageID)
	logger.Debug("SendMessageToIMSvr %v", &protocol.SSMessage{Head: head, Body: body})
	//return im.sendmsg(&SSMessage{Head: head, Body: body})
	return nil
}
