package main

import (
	"crypto/cipher"
	"gogame/base/util"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"

	"gogame/base/conf"
	redisCache "gogame/base/dao/redis"
	"gogame/base/logger"
	"gogame/base/network"
	. "gogame/common"
	. "gogame/errcode"
	. "gogame/protocol"
	"gogame/protocol/pb"
)

const (
	SESS_KEYEXCG       = 0x1  // 是否已经交换完毕KEY
	SESS_ENCRYPT       = 0x2  // 是否可以开始加密
	SESS_KICKED_OUT    = 0x4  // 踢掉
	SESS_AUTHORIZED    = 0x8  // 已授权访问
	SESS_REPLACED      = 0x10 // 登录替换
	SESS_GAMESVR_CLOSE = 0x20 // 服务器关闭
)

var clientMgr *ClientMgr

type ClientMgr struct {
	sync.RWMutex
	clients map[int64]*ClientTask
}
type ClientTask struct {
	*network.TCPConnection
	Encoder        cipher.Stream // 加密器
	Decoder        cipher.Stream // 解密器
	AccountID      int64         // 玩家ID(portal时已经从login中获得accountID)
	GSID           uint16        // 游戏服ID;(redis也有存)
	TSID           uint16        // 组队服务器ID
	Flag           int32         // 会话标记
	LastPacketTime int64         // 前一个包到达时间
}

func NewClientMgr() *ClientMgr {
	return &ClientMgr{
		clients: make(map[int64]*ClientTask),
	}
}

func ClientMgrGetMe() *ClientMgr {
	if clientMgr == nil {
		mgr := NewClientMgr()
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&clientMgr)), nil, unsafe.Pointer(mgr)) {
			mgr.init()
		}
	}
	return clientMgr
}
func (clientMgr *ClientMgr) Connect(con *network.TCPConnection) {
	client := &ClientTask{
		TCPConnection:  con,
		LastPacketTime: time.Now().Unix(),
	}
	con.Owner = client
}
func (clientMgr *ClientMgr) Disconnect(*network.TCPConnection) {}
func (clientMgr *ClientMgr) Receive(con *network.TCPConnection, data []byte) {
	client := con.Owner.(ClientTask)
	// 解密
	if client.Flag&SESS_ENCRYPT != 0 {
		client.Decoder.XORKeyStream(data, data)
	}
	csMsg := &CSMessage{}
	err := csMsg.UnMarshal(data)
	if err != nil {
		logger.Error("Protocol unmarshal msg error(%v)", err)
		client.SendErrToMe(EC_UnmarshalFail, 0)
		client.TerminateClient(SESS_KICKED_OUT)
		return
	}
	msgID := csMsg.Head.MessageID
	packetNo := csMsg.Head.PacketNo
	msgBody := csMsg.Body
	bContinue, msgBody, ret := client.HookClientMessage(msgID, packetNo, msgBody)
	if ret != nil {
		logger.Error("HookClientMessage(%v) error:%v", msgID, ret)
		client.SendErrToMe(ret, packetNo)
		client.TerminateClient(SESS_KICKED_OUT)
		return
	}
	if bContinue == true {
		destType := GetMessageDestByMsgID(msgID)
		switch destType {
		case EntityType_GameSvr:
			ret := client.gameServerForward(msgID, packetNo, msgBody)
			if ret != nil {
				logger.Error("service id:%v execute failed, error:%v", msgID, ret)
				client.SendErrToMe(ret, packetNo)
				client.TerminateClient(SESS_KICKED_OUT)
			}
		case EntityType_IM:
			ret := client.imServerForward(msgID, packetNo, msgBody)
			if ret != nil {
				logger.Error("send message to im fail %d", ret)
			}
		case EntityType_TeamSvr:
			ret := client.teamServerForward(msgID, packetNo, msgBody)
			if ret != nil {
				logger.Error("message id:0x%x send to teamserver execute failed, error:%v", msgID, ret)
			}
		default:
			logger.Error("service id:%d has no destType:%v to precedue", msgID, destType)
			client.SendErrToMe(EC_UnknownMessage, packetNo)
			client.TerminateClient(SESS_KICKED_OUT)
		}
	}
}
func (clientMgr *ClientMgr) init() {
	logger.Info("Portal client mgr Init!!")
	util.TimeInterval(time.Second*10, clientMgr.timeAction)
}

func (clientMgr *ClientMgr) Close() {
	logger.Info("Portal client mgr finished!!")
	clientMgr.RLock()
	defer clientMgr.RUnlock()
	for _, vClient := range clientMgr.clients {
		go vClient.TerminateClient(SESS_GAMESVR_CLOSE | SESS_KICKED_OUT)
	}
}

func (clientMgr *ClientMgr) timeAction() {
	var clients []*ClientTask
	nowTime := time.Now().Unix()
	clientMgr.RLock()
	for _, t := range clientMgr.clients {
		if nowTime-t.LastPacketTime > TCP_PING_TIMEOUT {
			clients = append(clients, t)
		}
	}
	clientMgr.RUnlock()
	for _, t := range clients {
		t.TerminateClient(SESS_KICKED_OUT)
		clientMgr.Remove(t)
		logger.Info("Player %v too long time %v not receive, disconnect network", t.AccountID, nowTime-t.LastPacketTime)
	}
}

func (clientMgr *ClientMgr) ServerClose(svrID uint16) {
	// 暂时只关注gamesvr关闭的情况
	clientMgr.RLock()
	var serverClients []*ClientTask
	for _, vClient := range clientMgr.clients {
		if vClient.GSID == svrID {
			serverClients = append(serverClients, vClient)
		}
	}
	clientMgr.RUnlock()
	for i := 0; i < len(serverClients); i++ {
		serverClients[i].TerminateClient(SESS_GAMESVR_CLOSE)
	}
}

func (clientMgr *ClientMgr) GetByID(accountID int64) *ClientTask {
	clientMgr.RLock()
	defer clientMgr.RUnlock()
	client := clientMgr.clients[accountID]
	return client
}
func (clientMgr *ClientMgr) Add(task *ClientTask) {
	if task.AccountID != INVALID_ID {
		clientMgr.Lock()
		defer clientMgr.Unlock()
		clientMgr.clients[task.AccountID] = task
	}
}
func (clientMgr *ClientMgr) Remove(client *ClientTask) {
	clientMgr.Lock()
	if t, ok := clientMgr.clients[client.AccountID]; ok {
		if t == client {
			delete(clientMgr.clients, client.AccountID)
		}
	}
	clientMgr.Unlock()
}

func (clientMgr *ClientMgr) BroadcastMsg(csMsg *CSMessage) {
	clientMgr.RLock()
	defer clientMgr.RUnlock()
	for _, v_client := range clientMgr.clients {
		v_client.SendCSMsg(csMsg)
	}
}

//发送消息到客户端
func (client *ClientTask) SendMessageToMe(msg proto.Message) error {
	bMsgBody, msgID, err := Marshal(msg)
	if err != nil {
		logger.Error("SendMessageToMe %d protobuf marshal fail %s", client.AccountID, err.Error())
		return EC_MarshalFail
	}
	csMsg := &CSMessage{}
	csMsg.Head.MessageID = msgID
	csMsg.Body = bMsgBody
	return client.SendCSMsg(csMsg)
}

func (client *ClientTask) SendCSMsg(csMsg *CSMessage) error {
	data, _ := csMsg.Marshal()
	if client.Flag&SESS_ENCRYPT != 0 {
		client.Encoder.XORKeyStream(data, data)
	}
	if ok := client.Send(data); !ok {
		return EC_NetworkFail
	}
	return nil
}

//踢人
func (client *ClientTask) TerminateClient(flag int32) {
	client.Flag |= flag
	time.Sleep(300 * time.Millisecond)
	client.Close()
}

//发送错误码给客户端
func (client *ClientTask) SendErrToMe(err error, packetNo uint32) {
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
	client.SendMessageToMe(noti)
}

func (client *ClientTask) Close() {
	client.Shutdown()
}

// 处理收到的客户端消息
func (client *ClientTask) HookClientMessage(msgID uint16, packetNo uint32, bMsgBody []byte) (bool, []byte, error) {
	var ret error
	switch msgID {
	case MsgID_MessageCharLoginReq:
		bMsgBody, ret = client.handlerLoginMsg(bMsgBody)
		if ret != nil {
			return false, bMsgBody, ret
		}
	case MsgID_MessagePing:
		client.handlePing(packetNo)
		return false, bMsgBody, nil
		// todo: 可以再此处理跨服，连接 teamsvr，snsserver 的消息
	default:
		logger.Debug("%d receive message %d", client.AccountID, msgID)
	}

	if !IsProtocolExist(msgID) {
		logger.Warn("Client send unknown msg(0x%x)", msgID)
		return false, bMsgBody, EC_UnknownMessage
	}
	return true, bMsgBody, nil
}

//处理来此服务器的消息
func (client *ClientTask) HookServerMessage(ssMsg *SSMessageBody) (bool, error) {
	return true, nil
}

//处理登录消息
func (client *ClientTask) handlerLoginMsg(bMsgBody []byte) ([]byte, error) {
	logReq := &pb.MessageCharLoginReq{}
	err := proto.Unmarshal(bMsgBody, logReq)
	if err != nil {
		logger.Error("protobuf unmarsh(%v) error(%v)", bMsgBody, err)
		return bMsgBody, EC_UnmarshalFail
	}
	if logReq.GetAccountID() <= 0 {
		logger.Error("handleLoginMsg fail as accountID(%v) invalid", logReq.GetAccountID())
		return bMsgBody, EC_ParameterInvalid
	}
	client.AccountID = logReq.GetAccountID()
	logger.Debug("handleLoginMsg player %d login server from addr %v", client.AccountID, client.RemoteAddr())
	// 如果存在旧的连接，则踢出旧的连接
	tokenWithTs, err := redisCache.GetAccountToken(client.AccountID)
	if err != nil {
		logger.Warn("handleLoginMsg player %d login server from addr %v redis error %s", client.AccountID, client.RemoteAddr(), err.Error())
		return bMsgBody, EC_TokenInvalid
	}
	tokenAndTs := strings.Split(tokenWithTs, "|")
	if len(tokenAndTs) != 2 {
		return bMsgBody, EC_TokenInvalid
	}
	if tokenAndTs[0] != logReq.GetUserToken() {
		logger.Warn("handleLoginMsg player %d token invalid redis:%s protocol:%s", client.AccountID, tokenAndTs[0], logReq.GetUserToken())
		return bMsgBody, EC_TokenInvalid
	}
	oldClient := ClientMgrGetMe().GetByID(client.AccountID)
	if oldClient != nil {
		if oldClient == client {
			return bMsgBody, nil
		}
		logger.Debug("handleLoginMsg player %d old session from addr %v was kicked by new session from addr %v", client.AccountID, oldClient.RemoteAddr(), client.RemoteAddr())
		oldClient.SendErrToMe(EC_AccountReplaced, 0)
		oldClient.TerminateClient(SESS_REPLACED)
	}
	// gamesvr负载均衡
	redisCon := redisCache.Pool(redisCache.RedisServerTypeCross).Get()
	defer redisCon.Close()
	gsID, err := redisCache.UserLoc(redisCon, client.AccountID)
	if err != nil && err != redis.ErrNil {
		logger.Error("handleLoginMsg get user loc account %d to server %d failed %s", client.AccountID, gsID, err.Error())
		return bMsgBody, EC_ServerBusy
	}
	if GameSvrAlive(gsID) == false {
		if err != redis.ErrNil {
			logger.Warn("Account %d still in gamesvr %v", client.AccountID, gsID)
		}
		gsID = DispatchGameSvr()
	}
	logger.Debug("handleLoginMsg dispatch account %d to server %d", client.AccountID, gsID)
	// GameSvr died
	if gsID == 0 {
		return bMsgBody, EC_ServerBusy
	}
	client.GSID = gsID
	ClientMgrGetMe().Add(client)
	remoteAddr := client.RemoteAddr().String()
	ipIdx := strings.Index(remoteAddr, ":")
	if ipIdx > 0 {
		ipAddr := remoteAddr[:ipIdx]
		logReq.ClientIP = ipAddr
		bMsgBody, err = proto.Marshal(logReq)
		if err != nil {
			logger.Error("handleLoginMsg %d marshal msg fail %s", client.AccountID, err.Error())
			return bMsgBody, EC_MarshalFail
		}
		logger.Debug("handleLoginMsg %d client ip is %s", client.AccountID, ipAddr)
	}
	return bMsgBody, nil
}

// 处理ping消息
func (client *ClientTask) handlePing(packetNo uint32) error {
	pong := &pb.MessagePong{
		Base: &pb.BaseNoti{
			PacketNo: packetNo,
			NowTime:  time.Now().Unix(),
		},
	}
	return client.SendMessageToMe(pong)
}
func (client *ClientTask) gameServerForward(msgID uint16, packetNo uint32, bMsgBody []byte) error {
	if client.GSID == INVALID_GAMESVR_ID {
		logger.Error("Gamesvr invalid(%v) fail, pls relogin", client.GSID)
		return EC_GamesvrError
	}

	ssMsg := &SSMessageBody{
		MessageID:    msgID,
		PacketNo:     packetNo,
		SrcAccountID: client.AccountID,
		DstAccountID: client.AccountID,
		GSID:         client.GSID,
		Body:         bMsgBody,
	}
	return SendMessageToRouter(EntityType_GameSvr, client.GSID, ssMsg)
}

// forward messages to team server
func (client *ClientTask) teamServerForward(msgID uint16, packetNo uint32, bMsgBody []byte) error {
	ssMsg := &SSMessageBody{
		MessageID:    msgID,
		PacketNo:     packetNo,
		SrcAccountID: client.AccountID,
		DstAccountID: client.AccountID,
		GSID:         client.GSID,
		Body:         bMsgBody,
	}
	return SendMessageToRouter(EntityType_TeamSvr, client.TSID, ssMsg)
}

// forward messages to im server
func (client *ClientTask) imServerForward(msgID uint16, packetNo uint32, bMsgBody []byte) error {
	ssMsg := &SSMessageBody{
		MessageID:    msgID,
		PacketNo:     packetNo,
		SrcAccountID: client.AccountID,
		DstAccountID: client.AccountID,
		GSID:         client.GSID,
		Body:         bMsgBody,
	}
	return SendMessageToIMSvr(ssMsg)
}

// 获取协议目的地址
// 根据协议范围段区分
func GetMessageDestByMsgID(msgID uint16) int32 {
	switch msgID & Message_ID_MASK {
	case Message_IM_Start:
		return EntityType_IM
	case Message_Team_Start:
		return EntityType_TeamSvr
	default:
		return EntityType_GameSvr
	}
}

func SendMessageToRouter(dstType int8, dstID uint16, ssMsg *SSMessageBody) error {
	cfg := conf.GameGateConf()
	head := &SSMessageHead{
		SSCmd:     SSCMD_MSG,
		TransType: TransType_ByKey,
		SrcType:   EntityType_Portal,
		DstType:   dstType,
		SrcID:     uint16(cfg.ServerID),
		DstID:     uint16(dstID),
	}
	body, err := ssMsg.Marshal()
	if err != nil {
		logger.Error("SendMessageToRouter dstType %d dstID %d fail", dstType, dstID)
		return EC_MarshalFail
	}
	msg := &SSMessage{Head: head, Body: body}
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

func SendMessageToIMSvr(ssMsg *SSMessageBody) error {
	im := GetIMByAccountID(ssMsg.DstAccountID)
	if im == nil {
		logger.Debug("SendMessageToIMSvr get im nil %d", ssMsg.DstAccountID)
		return EC_ParameterInvalid
	}
	cfg := conf.GameGateConf()
	head := &SSMessageHead{
		SSCmd:     SSCMD_MSG,
		TransType: TransType_ByKey,
		SrcType:   EntityType_Portal,
		DstType:   EntityType_IM,
		SrcID:     uint16(cfg.ServerID),
		DstID:     im.id,
	}
	body, err := ssMsg.Marshal()
	if err != nil {
		logger.Error("SendMessageToIMSvr dstID %d fail", im.id)
		return EC_MarshalFail
	}
	logger.Debug("SendMessageToIMSvr %d msg 0x%x", ssMsg.DstAccountID, ssMsg.MessageID)
	logger.Debug("SendMessageToIMSvr %v", &SSMessage{Head: head, Body: body})
	return im.SendMsg(&SSMessage{Head: head, Body: body})
}
