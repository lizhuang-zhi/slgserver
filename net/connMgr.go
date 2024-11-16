package net

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/llr104/slgserver/log"
	"github.com/llr104/slgserver/server/slgserver/conn"
	"github.com/llr104/slgserver/server/slgserver/pos"
	"go.uber.org/zap"
)

var ConnMgr = Mgr{}
var cid int64 = 0

type Mgr struct {
	cm sync.RWMutex
	um sync.RWMutex
	rm sync.RWMutex

	connCache map[int64]WSConn
	userCache map[int]WSConn
	roleCache map[int]WSConn
}

func (this *Mgr) NewConn(wsSocket *websocket.Conn, needSecret bool) *ServerConn {
	this.cm.Lock()
	defer this.cm.Unlock()

	cid++
	if this.connCache == nil {
		this.connCache = make(map[int64]WSConn)
	}

	if this.userCache == nil {
		this.userCache = make(map[int]WSConn)
	}

	if this.roleCache == nil {
		this.roleCache = make(map[int]WSConn)
	}

	c := NewServerConn(wsSocket, needSecret)
	c.SetProperty("cid", cid)
	this.connCache[cid] = c

	return c
}

func (this *Mgr) UserLogin(conn WSConn, session string, uid int) {
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
			oldConn.Push("robLogin", nil)
		}
	}
	this.userCache[uid] = conn
	conn.SetProperty("session", session)
	conn.SetProperty("uid", uid)
}

func (this *Mgr) UserLogout(conn WSConn) {
	this.removeUser(conn)
}

func (this *Mgr) removeUser(conn WSConn) {
	this.um.Lock()
	uid, err := conn.GetProperty("uid")
	if err == nil {
		//只删除自己的conn
		id := uid.(int)
		c, ok := this.userCache[id]
		if ok && c == conn {
			delete(this.userCache, id)
		}
	}
	this.um.Unlock()

	this.rm.Lock()
	rid, err := conn.GetProperty("rid")
	if err == nil {
		//只删除自己的conn
		id := rid.(int)
		c, ok := this.roleCache[id]
		if ok && c == conn {
			delete(this.roleCache, id)
		}
	}
	this.rm.Unlock()

	conn.RemoveProperty("session")
	conn.RemoveProperty("uid")
	conn.RemoveProperty("role")
	conn.RemoveProperty("rid")
}

func (this *Mgr) RoleEnter(conn WSConn, rid int) {
	this.rm.Lock()
	defer this.rm.Unlock()
	conn.SetProperty("rid", rid)
	this.roleCache[rid] = conn
}

func (this *Mgr) RemoveConn(conn WSConn) {
	this.cm.Lock()
	cid, err := conn.GetProperty("cid")
	if err == nil {
		delete(this.connCache, cid.(int64))
		conn.RemoveProperty("cid")
	}
	this.cm.Unlock()

	this.removeUser(conn)
}

func (this *Mgr) PushByRoleId(rid int, msgName string, data interface{}) bool {
	if rid <= 0 {
		return false
	}
	this.rm.Lock()
	defer this.rm.Unlock()
	conn, ok := this.roleCache[rid]
	if ok {
		conn.Push(msgName, data)
		return true
	} else {
		return false
	}
}

func (this *Mgr) Count() int {
	this.cm.RLock()
	defer this.cm.RUnlock()

	return len(this.connCache)
}

func (this *Mgr) Push(pushSync conn.PushSync) {

	proto := pushSync.ToProto()
	belongRIds := pushSync.BelongToRId()
	isCellView := pushSync.IsCellView()
	x, y := pushSync.Position()
	cells := make(map[int]int)

	//推送给开始位置
	if isCellView {
		cellRIds := pos.RPMgr.GetCellRoleIds(x, y, 8, 6)
		for _, rid := range cellRIds {
			//是否能出现在视野
			if can := pushSync.IsCanView(rid, x, y); can {
				this.PushByRoleId(rid, pushSync.PushMsgName(), proto)
				cells[rid] = rid
			}
		}
	}

	//推送给目标位置
	tx, ty := pushSync.TPosition()
	if tx >= 0 && ty >= 0 {
		var cellRIds []int
		if isCellView {
			cellRIds = pos.RPMgr.GetCellRoleIds(tx, ty, 8, 6)
		} else {
			cellRIds = pos.RPMgr.GetCellRoleIds(tx, ty, 0, 0)
		}

		for _, rid := range cellRIds {
			if _, ok := cells[rid]; ok == false {
				if can := pushSync.IsCanView(rid, tx, ty); can {
					this.PushByRoleId(rid, pushSync.PushMsgName(), proto)
					cells[rid] = rid
				}
			}
		}
	}

	//推送给自己
	for _, rid := range belongRIds {
		if _, ok := cells[rid]; ok == false {
			this.PushByRoleId(rid, pushSync.PushMsgName(), proto)
		}
	}

}

func (this *Mgr) pushAll(msgName string, data interface{}) {

	this.rm.Lock()
	defer this.rm.Unlock()
	for _, conn := range this.roleCache {
		conn.Push(msgName, data)
	}
}
