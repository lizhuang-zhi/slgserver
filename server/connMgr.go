package server

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"slgserver/log"
	"slgserver/net"
	"sync"
)

var DefaultConnMgr = ConnMgr{}
var cid = 0
type ConnMgr struct {
	cm        sync.RWMutex
	um        sync.RWMutex

	connCache map[int]*net.WSConn
	userCache map[int]*net.WSConn
}

func (this* ConnMgr) NewConn(wsSocket *websocket.Conn, needSecret bool) *net.WSConn{
	this.cm.Lock()
	defer this.cm.Unlock()

	cid++
	if this.connCache == nil {
		this.connCache = make(map[int]*net.WSConn)
	}

	if this.userCache == nil {
		this.userCache = make(map[int]*net.WSConn)
	}

	c := net.NewWSConn(wsSocket, needSecret)
	c.SetProperty("cid", cid)
	this.connCache[cid] = c

	return c
}

func (this* ConnMgr) UserLogin(conn *net.WSConn, session string, uid int) {
	this.um.Lock()
	defer this.um.Unlock()

	oldConn, ok := this.userCache[uid]
	if ok {
		if conn != oldConn {
			log.DefaultLog.Info("rob login",
				zap.Int("uid", uid),
				zap.String("oldAddr", oldConn.Addr()),
				zap.String("newAddr", conn.Addr()))

			//这里需要通知旧端被抢登录
			oldConn.Send("robLogin", nil)
		}
	}
	this.userCache[uid] = conn
	conn.SetProperty("session", session)
	conn.SetProperty("uid", uid)
}

func (this* ConnMgr) UserLogout(conn *net.WSConn) {
	this.um.Lock()
	defer this.um.Unlock()
	uid, err := conn.GetProperty("uid")
	if err == nil {
		delete(this.userCache, uid.(int))
	}
	conn.RemoveProperty("session")
	conn.RemoveProperty("uid")

}

func (this* ConnMgr) RemoveConn(conn *net.WSConn){
	this.cm.Lock()
	cid, err := conn.GetProperty("cid")
	if err == nil {
		delete(this.connCache, cid.(int))
	}
	this.cm.Unlock()

	this.um.Lock()
	uid, err := conn.GetProperty("uid")
	if err == nil {
		delete(this.userCache, uid.(int))
	}
	this.um.Unlock()
}

func (this* ConnMgr) Count() int{
	this.cm.RLock()
	defer this.cm.RUnlock()

	return len(this.connCache)
}