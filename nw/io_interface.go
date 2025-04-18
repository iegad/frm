package nw

import (
	"net"
)

// IService 引擎接口
//   - 服务端接口
type IService interface {
	// 会话连接事件
	//  - 当前客户端连接到 server时, 并成功创建会话时触发
	//  - 当该接口返回 err 时, 将主动关闭会话
	OnConnected(sess ISess) error

	// 会话连接断开事件
	//  - 当会话关闭时触发
	//  - 在该事件句柄框架会自动调用 sess.Close, 所以无需在该句柄中手动关闭会话
	OnDisconnected(sess ISess)

	// 接收数据事件
	//  - 当 read 到有效数据时触发
	//  - 当该接口返回 false 时, 将主动关闭会话
	OnData(sess ISess, data []byte) bool

	// 服务启动事件
	//  - 在创建监听对象后, 监听(Accept)之前触发
	//  - 当该接口返回 err 时, 服务将关闭
	OnStarted(ios *IoServer) error

	// 服务停止事件
	//  - 在服务收到停止请求后, 停止服务前触发
	OnStopped(ios *IoServer)
}

// ISess 引擎接口
//   - 服务端会话接口
type ISess interface {
	IsConnected() bool

	SockFd() int64

	// 本端地址
	LocalAddr() net.Addr

	// 远端地址
	RemoteAddr() net.Addr

	// 获取真实IP
	RealRemoteIP() string

	// 关闭会话
	Close() error

	// 发送数据
	Write(data []byte) (int, error)

	// 读取数据
	Read() ([]byte, error)

	// 获取用户自定义数据
	GetUserData() any

	// 设置用户自定义数据
	SetUserData(userData any)

	// 获取接收序号
	//	- 接收序号会在每次成功接收数据后 +1
	GetRecvSeq() int64

	// 获取发送序号
	//  - 发送序号会在每次成功发送数据后 +1
	GetSendSeq() int64
}

// IClient 引擎接口
//   - 客户端接口
type IClient ISess
