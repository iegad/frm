package nw

import (
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	sessPool = utils.NewPool[tcpSess]()
)

// IO服务配置
type IOSConfig struct {
	IP      string `yaml:"ip"         json:"ip,omitempty"`         // 服务监听地址
	TcpPort uint16 `yaml:"tcp_port"   json:"tcp_port,omitempty"`   // TCP 监听端口
	WsPort  uint16 `yaml:"ws_port"    json:"ws_port,omitempty"`    // websocket 监听端口
	Blend   uint32 `yaml:"blend"      json:"blend,omitempty"`      // 消息头混合值
	MaxConn uint32 `yaml:"max_conn"   json:"max_conn,omitempty"`   // 最大连接数, 最大连接数不能超过 int32的最大值, 否则无效
	Timeout uint32 `yaml:"timeout(s)" json:"timeout(s),omitempty"` // 客户端超时值
}

func (this_ *IOSConfig) String() string {
	return utils.ToJson(this_)
}

// IO服务
//   - 可同时监听 TCP 和 Websocket 两种协议, 但需要不同的两个端口
type IoServer struct {
	maxConn      int32                         // 最大连接数
	tcpHeadBlend uint32                        // TCP 消息头混合值
	tcpAddr      *net.TCPAddr                  // TCP 监听 Endpoint
	tcpListener  *net.TCPListener              // tcp listener
	wsAddr       *net.TCPAddr                  // websocket 监听 Endpoint
	wsListener   *net.TCPListener              // websocket listener
	timeout      time.Duration                 // 客户端超时
	service      IService                      // 服务实例
	mtx          sync.Mutex                    // 开启与停止互斥锁. 开启服务和停止服务存在并发, 所以需要互斥
	sessmap      *utils.SafeMap[string, ISess] // 会话集
	wg           sync.WaitGroup                // 协程同步组
	running      int32                         // 运行状态
}

// 创建io server 实例
func NewIOServer(cfg *IOSConfig, service IService) (*IoServer, error) {
	if cfg == nil {
		return nil, ErrConfigNil
	}

	if service == nil || reflect.ValueOf(service).IsNil() {
		return nil, ErrServiceNil
	}

	if cfg.TcpPort == 0 && cfg.WsPort == 0 {
		return nil, ErrNoListen
	}

	if cfg.Blend == 0 {
		return nil, ErrBlend
	}

	var (
		tcpAddr *net.TCPAddr
		wsAddr  *net.TCPAddr
		err     error
	)

	if cfg.TcpPort > 0 {
		tcpAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", cfg.IP, cfg.TcpPort))
		if err != nil {
			log.Error(err)
			return nil, ErrTcpEPInvalid
		}
	}

	if cfg.WsPort > 0 {
		wsAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", cfg.IP, cfg.WsPort))
		if err != nil {
			log.Error(err)
			return nil, ErrWsEPInvalid
		}
	}

	maxConn := cfg.MaxConn
	if maxConn >= math.MaxInt32 {
		maxConn = math.MaxInt32
	}

	this_ := &IoServer{
		tcpAddr:      tcpAddr,
		wsAddr:       wsAddr,
		tcpHeadBlend: cfg.Blend,
		tcpListener:  nil,
		wsListener:   nil,
		maxConn:      int32(maxConn),
		timeout:      time.Duration(cfg.Timeout) * time.Second,
		service:      service,
		mtx:          sync.Mutex{},
		sessmap:      utils.NewSafeMap[string, ISess](),
		wg:           sync.WaitGroup{},
		running:      0,
	}

	if this_.wsAddr != nil {
		http.HandleFunc("/ws", this_.wsUpgrade)
	}

	return this_, nil
}

// 获取 TCP 监听地址
func (this_ *IoServer) TcpAddr() *net.TCPAddr {
	if this_.tcpAddr != nil {
		return this_.tcpAddr
	}

	return nil
}

// 获取 websocket 监听地址
func (this_ *IoServer) WsAddr() *net.TCPAddr {
	if this_.wsAddr != nil {
		return this_.wsAddr
	}

	return nil
}

// 停止IO 服务
func (this_ *IoServer) Stop() {
	this_.mtx.Lock()
	defer this_.mtx.Unlock()

	if !atomic.CompareAndSwapInt32(&this_.running, 1, 0) {
		return
	}

	var err error

	if this_.tcpListener != nil {
		err = this_.tcpListener.Close()
		if err != nil {
			log.Error(err)
		}
		this_.tcpListener = nil
	}

	if this_.wsListener != nil {
		err = this_.wsListener.Close()
		if err != nil {
			log.Error(err)
		}
		this_.wsListener = nil
	}

	// 清理所有会话
	this_.sessmap.Range(func(remoteAddr string, sess ISess) bool {
		sess.Close()
		return true
	})

	this_.sessmap.Clear()

	// 服务会等待所有协程释放之后再返回
	this_.wg.Wait()
	this_.service.OnStopped(this_)
}

// 启动服务(阻塞)
func (this_ *IoServer) Run() error {
	err := this_.run()
	if err != nil {
		return err
	}

	this_.wg.Wait()
	return nil
}

// 启动服务
func (this_ *IoServer) run() error {
	this_.mtx.Lock()
	defer this_.mtx.Unlock()

	if !atomic.CompareAndSwapInt32(&this_.running, 0, 1) {
		return nil
	}

	var err error

	if this_.tcpAddr != nil {
		this_.tcpListener, err = net.ListenTCP("tcp", this_.tcpAddr)
	}

	if this_.wsAddr != nil {
		this_.wsListener, err = net.ListenTCP("tcp", this_.wsAddr)
	}

	if err != nil {
		if this_.tcpListener != nil {
			this_.tcpListener.Close()
		}

		if this_.wsListener != nil {
			this_.wsListener.Close()
		}

		return err
	}

	err = this_.service.OnStarted(this_)
	if err != nil {
		if this_.tcpListener != nil {
			this_.tcpListener.Close()
		}

		if this_.wsListener != nil {
			this_.wsListener.Close()
		}

		return err
	}

	if this_.tcpListener != nil {
		this_.wg.Add(1)
		go this_.tcpRun(&this_.wg)
	}

	if this_.wsListener != nil {
		this_.wg.Add(1)
		go this_.wsRun(&this_.wg)
	}

	return nil
}

// 启动 tcp 监听服务
func (this_ *IoServer) tcpRun(wg *sync.WaitGroup) {
	maxConn := this_.maxConn

	for {
		conn, err := this_.tcpListener.AcceptTCP()
		if err != nil {
			if IsClosedErr(err) {
				break
			}
			log.Error(err)
			continue
		}

		if maxConn > 0 && int32(this_.sessmap.Count()) >= maxConn {
			log.Error("TcpSess[%v] ACTIVE close. Error: Connection limit reached", conn.RemoteAddr())
			conn.Close()
			continue
		}

		wg.Add(1)
		go this_.tcpConnHandle(conn, wg)
	}

	wg.Done()
}

// tcp conn 句柄
func (this_ *IoServer) tcpConnHandle(conn *net.TCPConn, wg *sync.WaitGroup) {
	sess := sessPool.Get()
	err := sess.init(conn, this_.timeout, this_.tcpHeadBlend, this_.service)
	if err != nil {
		log.Error(err)
		conn.Close()
		return
	}

	var rbuf []byte

	defer func() {
		this_.sessmap.Remove(sess.RemoteAddr().String())
		sess.Close()
		this_.service.OnDisconnected(sess)
		sessPool.Put(sess)
		wg.Done()
	}()

	err = this_.service.OnConnected(sess)
	if err != nil {
		log.Error(err)
		return
	}

	this_.sessmap.Set(sess.RemoteAddr().String(), sess)

	for {
		rbuf, err = sess.Read()
		if err != nil {
			if err == io.EOF || IsConnReset(err) {
				log.Debug("TcpSess[%v] PASSIVE close", sess.RemoteAddr())
			} else {
				log.Error("TcpSess[%v] ACTIVE close. Error: [%T]%v", sess.RemoteAddr(), err, err)
			}
			break
		}

		if !this_.service.OnData(sess, rbuf) {
			break
		}
	}
}

// 启动 websocket 监听
func (this_ *IoServer) wsRun(wg *sync.WaitGroup) {
	err := http.Serve(this_.wsListener, nil)
	if err != nil {
		log.Error(err)
	}
	wg.Done()
}

// http 转换 websocket
func (this_ *IoServer) wsUpgrade(w http.ResponseWriter, r *http.Request) {
	if this_.maxConn > 0 && int32(this_.sessmap.Count()) >= this_.maxConn {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}

	this_.wg.Add(1)
	go this_.wsConnHandle(conn, &this_.wg, GetHttpRequestRealIP(r))
}

// websocket conn 句柄
func (this_ *IoServer) wsConnHandle(conn *websocket.Conn, wg *sync.WaitGroup, realIP string) {
	sess, err := newWsSess(conn, this_.timeout, realIP, this_.service)
	if err != nil {
		log.Error(err)
		conn.Close()
		return
	}

	var rbuf []byte

	defer func() {
		this_.sessmap.Remove(sess.RemoteAddr().String())
		sess.Close()
		this_.service.OnDisconnected(sess)
		wg.Done()
	}()

	err = this_.service.OnConnected(sess)
	if err != nil {
		log.Error(err)
		return
	}

	this_.sessmap.Set(sess.RemoteAddr().String(), sess)

	for {
		rbuf, err = sess.Read()
		if err != nil {
			if websocket.IsCloseError(err,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived) || IsConnReset(err) {
				log.Debug("WsSess[%v] PASSIVE close", sess.RemoteAddr())
			} else {
				log.Error("WsSess[%v] ACTIVE close. Error: %v", sess.RemoteAddr(), err)
			}
			break
		}

		if !this_.service.OnData(sess, rbuf) {
			break
		}
	}
}
