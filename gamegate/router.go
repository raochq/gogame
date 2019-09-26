package main

import (
	"github.com/golang/protobuf/proto"
	"gogame/base/logger"
	"gogame/base/network"
	"gogame/base/network/session"
	"gogame/base/util"
	. "gogame/common"
	. "gogame/protocol"
	"gogame/protocol/pb"
	"sync"
)

type Router struct {
	addr   string
	client *network.TCPClient
}
type RouterMgr struct {
	sync.Mutex
	routerMgr map[uint16]*Router
	routerMap map[string]bool
}

func NewRouterMgr() *RouterMgr {
	return &RouterMgr{
		routerMgr: make(map[uint16]*Router),
		routerMap: make(map[string]bool),
	}
}

func NewRouter(addr string) *Router {
	router := new(Router)
	router.addr = addr
	return router
}

func (r *Router) Start() {
	r.client = network.NewTCPClient(r.addr)
	r.client.DialAndServe(r, session.DefaultSessionCodec)
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
		clientMgr := GetGateServerInstance().clientMgr
		client := clientMgr.GetByID(ssMsg.DstAccountID)
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
		GetGateServerInstance().clientMgr.BroadcastMsg(csMsg)
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

func AddCrossRouterServer(ids []int32, router *Router) {
	// 每次都是全量同步
	svr := GetGateServerInstance().routerMgr
	oldsvrs := []int32{}
	svr.Lock()

	for svrID, v_router := range svr.routerMgr {
		if v_router == router {
			oldsvrs = append(oldsvrs, int32(svrID))
			delete(svr.routerMgr, svrID)
		}
	}
	for _, id := range ids {
		svr.routerMgr[uint16(id)] = router
	}
	svr.Unlock()
	logger.Debug("oldsvrs %v, new %v", oldsvrs, ids)
	for _, id := range oldsvrs {
		if !util.IsInt32InSlice(ids, id) {
			go GetGateServerInstance().clientMgr.ServerClose(uint16(id))
		}
	}
}
func AddRouterServer(ids []int32, r *Router) {

}
