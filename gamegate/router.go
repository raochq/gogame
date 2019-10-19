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
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var routerMgr *RouterMgr

type Router struct {
	addr   string
	client *network.TCPClient
}
type RouterMgr struct {
	sync.RWMutex
	routerMgr      map[uint16]*Router
	crossRouterMgr map[uint16]*Router
	routerMap      map[string]bool
	svrList        map[uint16]int //gamesvr状态列表
	teamIds        []uint16       //teamsvr列表
}

func NewRouterMgr() *RouterMgr {
	return &RouterMgr{
		routerMgr:      make(map[uint16]*Router),
		crossRouterMgr: make(map[uint16]*Router),
		routerMap:      make(map[string]bool),
		svrList:        make(map[uint16]int),
	}
}
func RouterMgrGetMe() *RouterMgr {
	if routerMgr == nil {
		mgr := NewRouterMgr()
		atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&routerMgr)), nil, unsafe.Pointer(mgr))
	}
	return routerMgr
}

func NewRouter(addr string) *Router {
	router := new(Router)
	router.addr = addr
	return router
}

func (routerMgr *RouterMgr) RouterInit() bool {
	cfg := conf.GameGateConf()
	routerAddrs := cfg.RouterAddrs
	if len(routerAddrs) <= 0 {
		logger.Error("Router init fail no routers")
		return false
	}
	for i := 0; i < len(routerAddrs); i++ {
		logger.Debug("RouterInit connect router %s", routerAddrs[i])
		routerMgr.routerMap[routerAddrs[i]] = true
		router := NewRouter(routerAddrs[i])
		router.Start()
	}

	crossRouterAddress := cfg.CrossRouterAddrs
	if len(crossRouterAddress) > 0 {
		for _, addr := range crossRouterAddress {
			logger.Debug("CrossRouterInit connect router %s", addr)
			routerMgr.routerMap[addr] = true
			router := NewRouter(addr)
			router.Start()
		}
	}
	return true
}

func (r *Router) Connect(con *network.TCPConnection) {
	logger.Debug("router connection router %s", con.RemoteAddr())
}

func (r *Router) Disconnect(con *network.TCPConnection) {
	logger.Debug("router disconnect %s", con.RemoteAddr())
}

func (r *Router) Receive(con *network.TCPConnection, data []byte) {
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
	switch msg.Head.SSCmd {
	case SSCMD_MSG:
		r.dispatch(msg.Head, body)
	case SSCMD_SYNC:
		r.sync(body)
	case SSCMD_HEARTBEAT:
		logger.Debug("router receive heartbeat")
	default:
		logger.Error("router receive unknown cmd: %d", msg.Head.SSCmd)
	}
}

func (r *Router) dispatch(head *SSMessageHead, ssMsg *SSMessageBody) {
	transType := head.TransType
	csMsg := &CSMessage{}
	csMsg.Head.MessageID = ssMsg.MessageID
	csMsg.Head.PacketNo = ssMsg.PacketNo
	csMsg.Body = ssMsg.Body
	if transType == TransType_ByKey {
		client := ClientMgrGetMe().GetByID(ssMsg.DstAccountID)
		if client == nil {
			// 玩家离线时，消息还未发完
			logger.Debug("dispatch get session fail for account %d", ssMsg.DstAccountID)
			return
		}
		if head.SrcType == EntityType_TeamSvr {
			if head.SrcID != client.TSID && client.TSID != 0 {
				logger.Warn("wrong TSID accountID %d from %d real %d", client.AccountID, head.SrcID, client.TSID)
			}
		}
		bContinue, err := client.HookServerMessage(ssMsg)
		if err == nil && bContinue == true {
			client.SendCSMsg(csMsg)
		}
	} else if transType == TransType_Broadcast {
		logger.Debug("Broadcast message 0x%x from account %d", ssMsg.MessageID, ssMsg.SrcAccountID)
		ClientMgrGetMe().BroadcastMsg(csMsg)
	} else {
		logger.Warn("Portal get message from router with unknown trans_type %d message 0x%x", transType, ssMsg.MessageID)
	}
}

func (r *Router) sync(ssmsg *SSMessageBody) {
	syncMsg := &pb.MessageSyncServer{}
	err := proto.Unmarshal(ssmsg.Body, syncMsg)
	if err != nil {
		logger.Error("sync message err %v", err)
		return
	}
	if syncMsg.GetServerType() == EntityType_TeamSvr {
		AddCrossRouterServer(syncMsg.GetServerIDs(), r)
	} else {
		AddRouterServer(syncMsg.GetServerIDs(), r)
	}
}

// 向Router注册
func (router *Router) register() error {
	head := &SSMessageHead{
		SSCmd:   SSCMD_READY,
		SrcType: EntityType_Portal,
		SrcID:   uint16(conf.GameGateConf().ServerID),
	}
	msg := &SSMessage{Head: head, Body: nil}
	data := PackSSMessage(msg)
	if data == nil {
		logger.Error("Register to router %v fail", head)
		return EC_MarshalFail
	}
	for {
		ok := router.client.GetConnection().Send(data)
		if !ok {
			break
		}
		time.Sleep(time.Second)
	}
	logger.Info("register srctype %d srcid %d", head.SrcType, head.SrcID)
	return nil
}

func (r *Router) Start() {
	r.client = network.NewTCPClient(r.addr)
	go r.client.DialAndServe(r, session.DefaultSessionCodec, 2*time.Second)
	go func() {
		for {
			ok := <-r.client.ConnectChan
			if ok == false {
				RemoveSvrs(r)
				return
			}
			r.register()
		}
	}()
}

func AddCrossRouterServer(svrIds []int32, router *Router) {
	// 每次都是全量同步
	svr := RouterMgrGetMe()
	var oldSvrIds []int32
	svr.Lock()

	for svrID, vRouter := range svr.crossRouterMgr {
		if vRouter == router {
			oldSvrIds = append(oldSvrIds, int32(svrID))
			delete(svr.crossRouterMgr, svrID)
		}
	}
	for _, id := range svrIds {
		svr.crossRouterMgr[uint16(id)] = router
	}
	var teamIds []uint16
	for teamId, _ := range svr.crossRouterMgr {
		teamIds = append(teamIds, teamId)
	}
	svr.teamIds = teamIds
	svr.Unlock()
	logger.Debug("update cross servers %v", svrIds)

}
func AddRouterServer(svrIds []int32, router *Router) {
	// 每次都是全量同步，先删除，插入
	svr := RouterMgrGetMe()
	var oldSvrIds []int32
	svr.Lock()
	for svrID, vRouter := range svr.routerMgr {
		if vRouter == router {
			oldSvrIds = append(oldSvrIds, int32(svrID))
			delete(svr.routerMgr, svrID)
		}
	}
	for _, id := range svrIds {
		svr.routerMgr[uint16(id)] = router
	}
	svr.Unlock()
	logger.Debug("old %v, new %v", oldSvrIds, svrIds)
	for _, id := range oldSvrIds {
		if !util.IsInt32InSlice(svrIds, id) {
			logger.Warn("server %d is closed", id)
			go ClientMgrGetMe().ServerClose(uint16(id))
		}
	}
}
func RemoveSvrs(router *Router) {
	svrMgr := RouterMgrGetMe()
	svrMgr.Lock()
	defer svrMgr.Unlock()

	for svrId, vRouter := range svrMgr.routerMgr {
		if vRouter == router {
			delete(svrMgr.routerMgr, svrId)
		}
	}
	var teamIds []uint16
	for svrId, vRouter := range svrMgr.crossRouterMgr {
		if vRouter == router {
			delete(svrMgr.crossRouterMgr, svrId)
			continue
		}
		teamIds = append(teamIds, svrId)
	}
	svrMgr.teamIds = teamIds
	delete(svrMgr.routerMap, router.addr)

}

func GameSvrAlive(serverId uint16) bool {
	svrMgr := RouterMgrGetMe()
	svrMgr.RLock()
	defer svrMgr.RUnlock()
	_, ok := svrMgr.svrList[serverId]
	return ok
}

func DispatchGameSvr() uint16 {
	svrMgr := RouterMgrGetMe()
	availSirs := make([]uint16, 0, len(svrMgr.svrList))
	svrMgr.RLock()
	for id, status := range svrMgr.svrList {
		if status == ServerStatusOpen {
			availSirs = append(availSirs, id)
		}
	}
	svrMgr.RUnlock()
	if len(availSirs) <= 0 {
		return 0
	}
	index := rand.Intn(len(availSirs))
	return availSirs[index]
}

func RefreshGameSvrList() {
	routerMgr = RouterMgrGetMe()
	now := time.Now().Unix()
	svrMap := redis.RefreshGameSvr()
	tSvrList := make(map[uint16]int)
	for id, data := range svrMap {
		mData := make(map[string]int)
		err := json.Unmarshal([]byte(data), &mData)
		if err != nil {
			logger.Warn("refresh portal list unmarshal error, %s", err.Error())
			continue
		}
		if mData["zoneid"] != conf.GameGateConf().ZoneID {
			continue
		}
		if int(now)-mData["timestamp"] > 15 {
			logger.Debug("game svr died, %s", id)
			continue
		}
		serverId, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		tSvrList[uint16(serverId)] = mData["status"]
	}
	routerMgr.Lock()
	routerMgr.svrList = tSvrList
	routerMgr.Unlock()
}
