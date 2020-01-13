package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
	"wisp/log"
)

type WebsocketConfig struct {
	websocketUrl          string                       //连接地址
	proxyUrl              string                       //代理地址
	requestHeaders        map[string][]string          //请求头设置map
	heartBeatIntervalTime time.Duration                //心跳周期
	heartBeatFunc         func() interface{}           //心跳函数
	heartBeatData         []byte                       //心跳数据
	reconnectIntervalTime time.Duration                //重连检测周期
	protocolHandleFunc    func([]byte) error           //协议处理
	unCompressFunc        func([]byte) ([]byte, error) //解压处理函数
	errorHandleFunc       func(error)                  //异常处理函数
	isDump                bool                         //是否开启堆栈信息
}

type WebsocketConnection struct {
	*websocket.Conn
	sync.Mutex
	WebsocketConfig
	activeTime     time.Time
	activeTimeL    sync.Mutex
	mu             chan struct{}
	closeHeartbeat chan struct{}
	closeReconnect chan struct{}
	closeRecv      chan struct{}
	closeCheck     chan struct{}
	subs           []interface{}
}

type WebsocketBuilder struct {
	*WebsocketConfig
}

func NewWebsocketBuilder() *WebsocketBuilder {
	return &WebsocketBuilder{&WebsocketConfig{}}
}

func (this *WebsocketBuilder) SetWebsocketUrl(url string) *WebsocketBuilder {
	this.websocketUrl = url
	return this
}

func (this *WebsocketBuilder) SetProxyUrl(proxy string) *WebsocketBuilder {
	this.proxyUrl = proxy
	return this
}

func (this *WebsocketBuilder) SetRequestHeaders(key, val string) *WebsocketBuilder {
	this.requestHeaders[key] = append(this.requestHeaders[key], val)
	return this
}

func (this *WebsocketBuilder) OpenDump() *WebsocketBuilder {
	this.isDump = true
	return this
}

func (this *WebsocketBuilder) SetHeartBeat(data []byte, t time.Duration) *WebsocketBuilder {
	this.heartBeatData = data
	this.heartBeatIntervalTime = t
	return this
}

func (this *WebsocketBuilder) SetHeartBeat1(event func() interface{}, t time.Duration) *WebsocketBuilder {
	this.heartBeatFunc = event
	this.heartBeatIntervalTime = t
	return this
}

func (this *WebsocketBuilder) SetReconnectIntervalTime(t time.Duration) *WebsocketBuilder {
	this.reconnectIntervalTime = t
	return this
}

func (this *WebsocketBuilder) SetProtocolHandle(handle func([]byte) error) *WebsocketBuilder {
	this.protocolHandleFunc = handle
	return this
}

func (this *WebsocketBuilder) SetUnCompressFunc(handle func(data []byte) ([]byte, error)) *WebsocketBuilder {
	this.unCompressFunc = handle
	return this
}

func (this *WebsocketBuilder) SetErrorHandle(handle func(error)) *WebsocketBuilder {
	this.errorHandleFunc = handle
	return this
}

func (this *WebsocketBuilder) Build() *WebsocketConnection {
	if this.errorHandleFunc == nil {
		this.errorHandleFunc = func(e error) {
			log.Info("异常信息: %v\n", e.Error())
		}
	}
	conn := &WebsocketConnection{
		WebsocketConfig: *this.WebsocketConfig,
	}
	return conn.New()
}

func (this *WebsocketConnection) New() *WebsocketConnection {
	this.Lock()
	defer this.Unlock()
	this.connect()
	this.mu = make(chan struct{}, 1)
	this.closeHeartbeat = make(chan struct{}, 1)
	this.closeReconnect = make(chan struct{}, 1)
	this.closeRecv = make(chan struct{}, 1)
	this.closeCheck = make(chan struct{}, 1)
	this.HeartbeatTimer()
	this.ReconnectTimer()
	this.checkStatusTimer()
	return this
}

func (this *WebsocketConnection) RecvMsg() {
	this.clearChannel(this.closeRecv)
	go func() {
		for {
			if len(this.closeRecv) > 0 {
				<-this.closeRecv
				log.Info("关闭连接,退出 RecvMsg协程\n")
				return
			}

			t, msg, err := this.ReadMessage()
			if err != nil {
				this.errorHandleFunc(err)
				time.Sleep(time.Second)
				continue
			}

			switch t {
			case websocket.TextMessage:
				this.protocolHandleFunc(msg)
			case websocket.BinaryMessage:
				if this.unCompressFunc == nil {
					this.protocolHandleFunc(msg)
				} else {
					nmsg, err := this.unCompressFunc(msg)
					if err != nil {
						this.errorHandleFunc(fmt.Errorf("%s,%s", "消息解压失败", err.Error()))
					} else {
						err := this.protocolHandleFunc(nmsg)
						if err != nil {
							this.errorHandleFunc(err)
						}
					}
				}
			case websocket.CloseMessage:
				this.Close()
				return
			default:
				log.Info("消息解析错误: %v %v\n", string(msg), err.Error())
			}
		}
	}()
}

func (this *WebsocketConnection) HeartbeatTimer() {
	log.Info("心跳周期时间为: %v\n", this.heartBeatIntervalTime)
	if this.heartBeatIntervalTime == 0 || (this.heartBeatFunc == nil && this.heartBeatData == nil) {
		return
	}

	timer := time.NewTicker(this.heartBeatIntervalTime)
	go func() {
		this.clearChannel(this.closeHeartbeat)
		for {
			select {
			case <-timer.C:
				var err error
				if this.heartBeatFunc != nil {
					err = this.SendJson(this.heartBeatFunc)
				} else {
					err = this.SendText(this.heartBeatData)
				}

				if err != nil {
					log.Info("心跳数据发送错误: %v\n", err.Error())
					time.Sleep(time.Second)
				}
			case <-this.closeHeartbeat:
				timer.Stop()
				log.Info("关闭websocket连接,退出心跳协程\n")
				return
			}
		}
	}()
}

func (this *WebsocketConnection) checkStatusTimer() {
	if this.heartBeatIntervalTime == 0 {
		return
	}
	timer := time.NewTimer(this.heartBeatIntervalTime)

	go func() {
		select {
		case <-timer.C:
			now := time.Now()
			if now.Sub(this.activeTime) >= 2*this.heartBeatIntervalTime {
				log.Info("上次一活动时间为: [%v],已经过期，开始重新连接\n", this.activeTime)
				this.Reconnect()
			}
			timer.Reset(this.heartBeatIntervalTime)
		case <-this.closeCheck:
			log.Info("退出状态监测协程\n")
			return
		}
	}()
}

func (this *WebsocketConnection) ReconnectTimer() {
	if this.reconnectIntervalTime == 0 {
		return
	}
	timer := time.NewTimer(this.reconnectIntervalTime)

	go func() {
		this.clearChannel(this.closeReconnect)
		for {
			select {
			case <-timer.C:
				log.Info("开始重新连接\n")
				this.Reconnect()
				timer.Reset(this.reconnectIntervalTime)
			case <-this.closeReconnect:
				timer.Stop()
				log.Info("关闭连接,退出当前协程\n")
				return
			}
		}
	}()
}

func (this *WebsocketConnection) Reconnect() {
	this.Lock()
	defer this.Unlock()
	this.Close()
	time.Sleep(time.Second)
	this.connect()
	for _, event := range this.subs {
		log.Info("订阅频道: %v\n", event)
		this.SendJson(event)
	}
}

func (this *WebsocketConnection) SendText(data []byte) error {
	this.mu <- struct{}{}
	defer func() {
		<-this.mu
	}()
	return this.WriteMessage(websocket.TextMessage, data)
}

func (this *WebsocketConnection) SendJson(data interface{}) error {
	this.mu <- struct{}{}
	defer func() {
		<-this.mu
	}()

	return this.WriteJSON(data)
}

func (this *WebsocketConnection) Subscribe(event interface{}) error {
	err := this.SendJson(event)
	if err != nil {
		return err
	}
	this.subs = append(this.subs, event)
	return nil
}

func (this *WebsocketConnection) connect() {
	dial := websocket.DefaultDialer
	if this.proxyUrl != "" {
		proxy, err := url.Parse(this.proxyUrl)
		if err != nil {
			log.Info("代理地址设置错误: 代理地址为: [%s] 错误信息: %v\n", this.proxyUrl, err.Error())
		} else {
			dial.Proxy = http.ProxyURL(proxy)
			//log.Info("设置当前连接代理地址为: [%s]\n", this.proxyUrl)
		}
	}

	conn, res, err := dial.Dial(this.websocketUrl, http.Header(this.requestHeaders))
	if err != nil {
		log.Info("连接发生错误: %v\n", err.Error())
		panic(err)
	}
	this.Conn = conn

	if this.isDump {
		dumpData, _ := httputil.DumpResponse(res, true)
		log.Info("连接堆栈信息: %v\n", string(dumpData))
	}
	this.UpdateActiveTime()
}

func (this *WebsocketConnection) UpdateActiveTime() {
	this.activeTimeL.Lock()
	defer this.activeTimeL.Unlock()
	this.activeTime = time.Now()
}

func (this *WebsocketConnection) clearChannel(c chan struct{}) {
	for {
		if len(c) > 0 {
			<-c
		} else {
			break
		}
	}
}
