package controller

import (
	"strings"
	"sync"

	"github.com/goinggo/mapstructure"
	"github.com/llr104/slgserver/config"
	"github.com/llr104/slgserver/constant"
	"github.com/llr104/slgserver/log"
	"github.com/llr104/slgserver/middleware"
	"github.com/llr104/slgserver/net"
	chat_proto "github.com/llr104/slgserver/server/chatserver/proto"
	"github.com/llr104/slgserver/server/slgserver/proto"
	"go.uber.org/zap"
)

var GHandle = Handle{
	proxys: make(map[string]map[int64]*net.ProxyClient),
}

type Handle struct {
	proxyMutex sync.Mutex
	proxys     map[string]map[int64]*net.ProxyClient
	slgProxy   string
	chatProxy  string
	loginProxy string
}

func isAccount(msgName string) bool {
	sArr := strings.Split(msgName, ".")
	prefix := ""
	if len(sArr) == 2 {
		prefix = sArr[0]
	}
	if prefix == "account" {
		return true
	} else {
		return false
	}
}

func isChat(msgName string) bool {
	sArr := strings.Split(msgName, ".")
	prefix := ""
	if len(sArr) == 2 {
		prefix = sArr[0]
	}
	if prefix == "chat" {
		return true
	} else {
		return false
	}
}

func (this *Handle) InitRouter(r *net.Router) {
	this.init()
	// 添加中间件
	g := r.Group("*").Use(middleware.ElapsedTime(), middleware.Log())
	g.AddRouter("*", this.all)
}

func (this *Handle) init() {
	this.slgProxy = config.File.MustValue("gateserver", "slg_proxy", "ws://127.0.0.1:8001")
	this.chatProxy = config.File.MustValue("gateserver", "chat_proxy", "ws://127.0.0.1:8002")
	this.loginProxy = config.File.MustValue("gateserver", "login_proxy", "ws://127.0.0.1:8003")
}

func (this *Handle) onPush(conn *net.ClientConn, body *net.RspBody) {
	gc, err := conn.GetProperty("gateConn")
	if err != nil {
		return
	}
	gateConn := gc.(net.WSConn)
	gateConn.Push(body.Name, body.Msg)
}

func (this *Handle) onProxyClose(conn *net.ClientConn) {
	p, err := conn.GetProperty("proxy")
	if err == nil {
		proxyStr := p.(string)
		this.proxyMutex.Lock()
		_, ok := this.proxys[proxyStr]
		if ok {
			c, err := conn.GetProperty("cid")
			if err == nil {
				cid := c.(int64)
				delete(this.proxys[proxyStr], cid)
			}
		}
		this.proxyMutex.Unlock()
	}
}

func (this *Handle) OnServerConnClose(conn net.WSConn) {
	c, err := conn.GetProperty("cid")
	arr := make([]*net.ProxyClient, 0)

	if err == nil {
		cid := c.(int64)
		this.proxyMutex.Lock()
		for _, m := range this.proxys {
			proxy, ok := m[cid]
			if ok {
				arr = append(arr, proxy)
			}
			delete(m, cid)
		}
		this.proxyMutex.Unlock()
	}

	for _, client := range arr {
		client.Close()
	}

}

func (this *Handle) all(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	// 记录日志，表示处理开始
	log.DefaultLog.Info("gateserver handle all begin",
		zap.String("proxyStr", req.Body.Proxy),
		zap.String("msgName", req.Body.Name))

	// 调用deal方法处理请求和响应
	this.deal(req, rsp)

	// 如果请求的消息名称是"role.enterServer"且响应的状态码是OK
	if req.Body.Name == "role.enterServer" && rsp.Body.Code == constant.OK {
		// 解码响应消息到rspObj
		rspObj := &proto.EnterServerRsp{}
		mapstructure.Decode(rsp.Body.Msg, rspObj)

		// 创建登录请求
		r := &chat_proto.LoginReq{RId: rspObj.Role.RId, NickName: rspObj.Role.NickName, Token: rspObj.Token}
		// 创建请求体
		reqBody := &net.ReqBody{Seq: 0, Name: "chat.login", Msg: r, Proxy: ""}
		// 创建响应体
		rspBody := &net.RspBody{Seq: 0, Name: "chat.login", Msg: r, Code: 0}
		// 再次调用deal方法处理登录请求和响应
		this.deal(&net.WsMsgReq{Body: reqBody, Conn: req.Conn}, &net.WsMsgRsp{Body: rspBody})
	}

	// 记录日志，表示处理结束
	log.DefaultLog.Info("gateserver handle all end",
		zap.String("proxyStr", req.Body.Proxy),
		zap.String("msgName", req.Body.Name))
}

// 转发请求到对应的服务
func (this *Handle) deal(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	// 协议转发
	// 获取请求体中的代理字符串
	proxyStr := req.Body.Proxy
	if isAccount(req.Body.Name) { // 判断msgNamae是否为acount.xxx(判断是否转发到login服务器)
		proxyStr = this.loginProxy
	} else if isChat(req.Body.Name) { // 判断msgName是否为chat.xxx(判断是否转发到聊天服务器)
		proxyStr = this.chatProxy
	} else {
		// 转发到slg服务器
		proxyStr = this.slgProxy
	}

	// 如果代理字符串为空，则返回错误代码
	if proxyStr == "" {
		rsp.Body.Code = constant.ProxyNotInConnect
		return
	}

	this.proxyMutex.Lock()
	// 判断代理字符串是否存在于代理映射中
	_, ok := this.proxys[proxyStr]
	if ok == false {
		// 如果不存在，则创建新的映射
		this.proxys[proxyStr] = make(map[int64]*net.ProxyClient)
	}

	// 声明错误变量和代理客户端变量
	var err error
	var proxy *net.ProxyClient
	// 从请求连接中获取cid属性
	d, _ := req.Conn.GetProperty("cid")
	cid := d.(int64)
	// 从代理映射中获取代理客户端
	proxy, ok = this.proxys[proxyStr][cid]
	this.proxyMutex.Unlock()

	// 如果代理客户端不存在
	if ok == false {
		// 创建新的代理客户端
		proxy = net.NewProxyClient(proxyStr)

		this.proxyMutex.Lock()
		this.proxys[proxyStr][cid] = proxy
		this.proxyMutex.Unlock()

		err = proxy.Connect()
		if err == nil {
			// 设置代理客户端的属性
			proxy.SetProperty("cid", cid)
			proxy.SetProperty("proxy", proxyStr)
			proxy.SetProperty("gateConn", req.Conn)
			proxy.SetOnPush(this.onPush)
			proxy.SetOnClose(this.onProxyClose)
		}
	}

	// 如果发生错误
	if err != nil {
		// 加锁
		this.proxyMutex.Lock()
		// 从代理映射中删除cid对应的代理客户端
		delete(this.proxys[proxyStr], cid)
		// 解锁
		this.proxyMutex.Unlock()
		// 设置响应体的错误代码
		rsp.Body.Code = constant.ProxyConnectError
		return
	}

	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name

	// 此时将gate网关看作客户端, 请求对应的服务
	r, err := proxy.Send(req.Body.Name, req.Body.Msg)
	if err == nil {
		rsp.Body.Code = r.Code
		rsp.Body.Msg = r.Msg
	} else {
		// 设置响应体的错误代码和消息为空
		rsp.Body.Code = constant.ProxyConnectError
		rsp.Body.Msg = nil
	}
}
