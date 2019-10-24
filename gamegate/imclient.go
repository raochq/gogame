package main

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"gogame/base/conf"
	"gogame/base/dao/redis"
	"gogame/base/logger"
	"gogame/base/network"
	"gogame/base/network/session"
	"gogame/base/util"
	. "gogame/common"
	. "gogame/errcode"
	. "gogame/protocol"
	"gogame/protocol/pb"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var imMgr *IMMgr

type IMMgr struct {
	sync.RWMutex
	ims      map[int]string
	imMap    map[uint16]*IMSvr
	hashRing *util.HashRing
	DieChan  chan struct{}
}

type IMSvr struct {
	*network.TCPClient

	addr   string
	id     uint16
	status int
}

func IMMgrGetMe() *IMMgr {
	if imMgr == nil {
		mgr := &IMMgr{
			ims:     make(map[int]string),
			imMap:   make(map[uint16]*IMSvr),
			DieChan: make(chan struct{}),
		}
		atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&imMgr)), nil, unsafe.Pointer(mgr))
	}
	return imMgr
}

func NewIMSvr(id uint16, addr string) *IMSvr {
	return &IMSvr{
		addr: addr,
		id:   id,
	}
}

func GetIMByAccountID(accountID int64) *IMSvr {
	imMgr := IMMgrGetMe()
	if imMgr.hashRing == nil {
		logger.Debug("GetIMByAccountID %d hashring nil")
		return nil
	}
	imMgr.RLock()
	defer imMgr.RUnlock()
	sid, ok := imMgr.hashRing.GetNode(strconv.Itoa(int(accountID)))
	if !ok {
		logger.Info("GetIMByAccountID %d get im node fail", accountID)
		return nil
	}
	imID, _ := strconv.Atoi(sid)
	return imMgr.imMap[uint16(imID)]
}

func GetIMByID(imid uint16) *IMSvr {
	imMgr := IMMgrGetMe()
	imMgr.RLock()
	defer imMgr.RUnlock()
	return imMgr.imMap[imid]
}

func refreshIMSvrList() {
	mgr := IMMgrGetMe()
	now := time.Now().Unix()

	svrMap := redis.RefreshIMSvr()
	newSvrlist := make(map[int]string)
	for id, data := range svrMap {
		mData := make(map[string]interface{})
		err := json.Unmarshal([]byte(data), &mData)
		if err != nil {
			logger.Warn("refresh portal list unmarshal error, %s", err.Error())
			continue
		}
		if now-int64(mData["timestamp"].(float64)) > 15 {
			logger.Debug("im svr died, %s", id)
			continue
		}
		serverId, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		newSvrlist[serverId] = mData["addr"].(string)
	}
	// add nodes
	addNodes := []int{}
	for sid, addr := range newSvrlist {
		if _, ok := imMgr.ims[sid]; !ok {
			// 启动
			mgr.Lock()
			im := NewIMSvr(uint16(sid), addr)
			im.Start()
			imMgr.imMap[uint16(sid)] = im
			mgr.Unlock()
			addNodes = append(addNodes, sid)
		}
	}
	if len(addNodes) > 0 {
		if imMgr.hashRing == nil {
			sNodes := map[string]int{}
			for i := 0; i < len(addNodes); i++ {
				sNodes[strconv.Itoa(addNodes[i])] = 1
				imMgr.ims[addNodes[i]] = ""
			}
			imMgr.hashRing = util.NewHashringWithWeights(sNodes)
		} else {
			for i := 0; i < len(addNodes); i++ {
				// 新增节点
				imMgr.Lock()
				imMgr.hashRing = imMgr.hashRing.AddNode(strconv.Itoa(addNodes[i]))
				imMgr.ims[addNodes[i]] = ""
				imMgr.Unlock()
			}
		}
	}
}
func (imMgr *IMMgr) Close() {
	close(imMgr.DieChan)
}
func (im *IMSvr) Start() {
	go im.run()

}
func (im *IMSvr) run() {
	for {
		im.TCPClient = network.NewTCPClient(im.addr)
		go im.DialAndServe(im, session.DefaultSessionCodec, 2*time.Second)
		ok := <-im.ConnectChan
		if !ok {
			continue
		}

		im.register()
		secTimer := time.NewTimer(time.Second)
		DieChan := IMMgrGetMe().DieChan
	loop:
		for {
			select {
			case <-secTimer.C:
				if im.IsClosed() == true {
					logger.Debug("IMSvr %d connection closed %v->%v ", im.id, im.GetConnection().LocalAddr(), im.GetConnection().RemoteAddr())
					secTimer.Stop()
					break loop
				}
			case <-DieChan:
				logger.Debug("IMSvr %d goodbye to cruel world", im.id)
				return
			}
		}
	}
}
func (im *IMSvr) Connect(con *network.TCPConnection) {
	logger.Debug("IMSvr connection router %s", con.RemoteAddr())
}

func (im *IMSvr) Disconnect(con *network.TCPConnection) {
	logger.Debug("IMSvr disconnect %s", con.RemoteAddr())
}
func (im *IMSvr) Receive(con *network.TCPConnection, data []byte) {
	msg := UnpackSSMessage(data)
	if msg == nil {
		return
	}

	body := &SSMessageBody{}
	err := body.UnMarshal(msg.Body)
	if err != nil {
		logger.Error("router receive unmarshal body error")
		return
	}
	head := msg.Head
	switch head.SSCmd {
	case SSCMD_READY:
		if head.SrcType == EntityType_IM {
			im.id = head.SrcID
			im.status = 1
			logger.Debug("im SvrType %d svrID is %d", head.SrcType, head.SrcID)
		} else {
			logger.Error("im SvrType %d svrID %d is not im server", head.SrcType, head.SrcID)
		}
	case SSCMD_MSG:
		msgBody := &SSMessageBody{}
		err := msgBody.UnMarshal(msg.Body)
		if err != nil {
			logger.Warn("im ssMessage Body err %s", err.Error())
			//im.Close()
		}
		im.dispatch(head, msgBody)
	case SSCMD_HEARTBEAT:
		logger.Debug("portal recv heartbeat from im")
	default:
		logger.Error("unknown message cmd: %d", head.SSCmd)
	}
}

// 向IMSvr注册
func (im *IMSvr) register() error {
	head := &SSMessageHead{
		SSCmd:   SSCMD_READY,
		SrcType: EntityType_Portal,
		SrcID:   uint16(conf.GameGateConf().ServerID),
	}
	msg := &SSMessage{Head: head, Body: nil}
	if err := im.sendMsg(msg); err != nil {
		logger.Error("Register to router %v fail", head)
		return EC_MarshalFail
	}
	return nil
}

// 关闭IMSvr
func (im *IMSvr) close(conn *net.TCPConn) error {
	head := &SSMessageHead{
		SSCmd:   SSCMD_DISCONNECT,
		SrcType: EntityType_Portal,
		SrcID:   uint16(conf.GameGateConf().ServerID),
	}
	msg := &SSMessage{Head: head, Body: nil}
	if err := im.sendMsg(msg); err != nil {
		logger.Error("Disconnect to im %v fail", head)
		return err
	}
	im.status = 0
	return nil
}
func (im *IMSvr) sendMsg(ssMsg *SSMessage) error {
	data := PackSSMessage(ssMsg)
	if data == nil {
		logger.Error("close to router %v fail", ssMsg.Head)
		return EC_MarshalFail
	}
	ok := im.GetConnection().Send(data)

	if !ok {
		logger.Error("Disconnect to im %v fail", ssMsg.Head)
		return EC_IMSvrNotFound
	}
	return nil
}
func (im *IMSvr) SendMsg(ssMsg *SSMessage) error {
	if im.status == 0 {
		return EC_IMSvrNotFound
	}
	return im.sendMsg(ssMsg)
}

// IMSvr分发消息
func (im *IMSvr) dispatch(head *SSMessageHead, ssMsg *SSMessageBody) error {
	transType := head.TransType
	if head.DstType == EntityType_Portal {
		// 发送到本portal的
		if transType == TransType_ByKey {
			client := ClientMgrGetMe().GetByID(ssMsg.DstAccountID)
			if client == nil {
				//im不做玩家是否在线检查，可能发送的目标玩家不在线
				//logger.Warn("dispatch get session fail for account %d", ssMsg.DstAccountID)
				return EC_PlayerNoOnline
			}
			bContinue, err := client.HookServerMessage(ssMsg)
			if err == nil && bContinue == true {
				csMsg := &CSMessage{}
				csMsg.Head.MessageID = ssMsg.MessageID
				csMsg.Body = ssMsg.Body
				client.SendCSMsg(csMsg)
			}
		} else if transType == TransType_Broadcast {
			logger.Debug("Broadcast message 0x%x from account %d", ssMsg.MessageID, ssMsg.SrcAccountID)
			csMsg := &CSMessage{}
			csMsg.Head.MessageID = ssMsg.MessageID
			csMsg.Body = ssMsg.Body
			ClientMgrGetMe().BroadcastMsg(csMsg)
		} else if transType == TransType_Group {
			if ssMsg.MessageID != uint16(MsgID_MessageGroupMsgNoti) {
				logger.Error("Portal recv TransType_Group but msg id is 0x%x not MsgID_MessageGroupMsgNoti", ssMsg.MessageID)
				return EC_ParameterInvalid
			}
			noti := &pb.MessageGroupMsgNoti{}
			err := proto.Unmarshal(ssMsg.Body, noti)
			if err != nil {
				logger.Error("Portal recv TransType_Group unmarshal fail %s", err.Error())
				return EC_UnmarshalFail
			}
			accounts := noti.GetAccountIDs()
			csMsg := &CSMessage{
				Head: CSMessageHead{MessageID: uint16(noti.GetMsgID())},
				Body: noti.GetMsgBody(),
			}
			for i := 0; i < len(accounts); i++ {
				client := ClientMgrGetMe().GetByID(accounts[i])
				if client == nil {
					//logger.Warn("dispatch get session fail for account %d", ssMsg.DstAccountID)
					continue
				}
				client.SendCSMsg(csMsg)
			}
		} else {
			logger.Warn("Portal get message from im with unknown trans_type %d message 0x%x", transType, ssMsg.MessageID)
			return EC_ParameterInvalid
		}
	} else if head.DstType == EntityType_IM {
		// 发送到其他IM的
		if transType == TransType_ByKey {
			dstID := head.DstID
			im := GetIMByID(dstID)
			if im == nil {
				logger.Warn("dispatch im server get im %d fail", dstID)
				return EC_IMSvrNotFound
			}
			head.SrcType = EntityType_Portal
			head.SrcID = uint16(conf.GameGateConf().ServerID)
			body, err := ssMsg.Marshal()
			if err != nil {
				logger.Warn("dispatch SSMessageBody marshal fail %s", err.Error())
				return EC_MarshalFail
			}
			return im.SendMsg(&SSMessage{Head: head, Body: body})
		} else if transType == TransType_Broadcast {
			body, err := ssMsg.Marshal()
			if err != nil {
				logger.Warn("dispatch SSMessageBody marshal fail %s", err.Error())
				return EC_MarshalFail
			}
			for _, imsvr := range imMgr.imMap {
				if head.SrcType == EntityType_IM &&
					imsvr.id == head.SrcID {
					// 源服务器上已经处理，无需广播
					continue
				}
				head := &SSMessageHead{
					SSCmd:     SSCMD_MSG,
					SrcType:   head.SrcType,
					SrcID:     head.SrcID,
					DstType:   EntityType_IM,
					DstID:     imsvr.id,
					TransType: TransType_ByKey,
				}
				imsvr.SendMsg(&SSMessage{Head: head, Body: body})
			}
		} else {
			logger.Warn("Portal get message from im with unknown trans_type %d message 0x%x", transType, ssMsg.MessageID)
			return EC_ParameterInvalid
		}
	} else {
		logger.Warn("dispatch message dsttype %d dst id %d is invalid", head.DstType, head.DstID)
		return EC_ParameterInvalid
	}

	return nil
}
