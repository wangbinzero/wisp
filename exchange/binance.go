package exchange

import (
	"errors"
	"fmt"
	"github.com/json-iterator/go"
	"strings"
	"sync"
	"time"
	. "wisp/common"
	"wisp/log"
	. "wisp/utils"
	"wisp/ws"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type binanceExchange struct {
	*ws.WebsocketBuilder
	sync.Once
	*ws.WebsocketConnection
	baseUrl         string
	combinedBaseUrl string //组合订阅地址
	depthCallback   func(*Depth)
	klineCallback   func(*Kline, int)
	tickerCallback  func(*Ticker)
}

func NewBinanceExchange() *binanceExchange {
	binance := &binanceExchange{WebsocketBuilder: ws.NewWebsocketBuilder()}
	binance.baseUrl = "wss://stream.binance.com:9443/ws"
	binance.combinedBaseUrl = "wss://stream.binance.com/stream?streams="
	return binance
}

func (this *binanceExchange) connect() {
	this.Do(func() {
		this.WebsocketConnection = this.WebsocketBuilder.Build()
		this.WebsocketConnection.RecvMsg()
	})
}

func (this *binanceExchange) protocolHandle(data []byte) error {
	text := string(data)
	log.Info("币安消息: %v\n", text)
	return nil
}

func (this *binanceExchange) errorHandle(err error) {
	log.Info("币安异常信息: %v\n", err.Error())
	this.Reconnect()
}

func (this *binanceExchange) subscribe(endpoint string, handle func(msg []byte) error) {
	builder := ws.NewWebsocketBuilder().
		SetWebsocketUrl(endpoint).
		SetReconnectIntervalTime(12 * time.Hour).
		SetProtocolHandle(handle).
		SetProxyUrl("socks5://127.0.0.1:1080").
		SetErrorHandle(this.errorHandle)
	conn := builder.Build()
	conn.RecvMsg()
}

func (this *binanceExchange) SubDepths(symbol string, size int) error {
	if this.depthCallback == nil {
		return errors.New("深度回调方法未初始化")
	}
	if size != 5 && size != 10 && size != 20 {
		return errors.New("深度订阅错误，超出档数: 5/10/20")
	}
	endpoint := fmt.Sprintf("%s%s@depth%d@1000ms", this.combinedBaseUrl, strings.ToLower(symbol), size)
	log.Info("打印深度端点: %s\n", endpoint)
	handle := func(msg []byte) error {
		rawDepth := struct {
			Stream string `json:"stream"`
			Data   struct {
				LastUpdateID int64           `json:"lastUpdateId"`
				Bids         [][]interface{} `json:"bids"`
				Asks         [][]interface{} `json:"asks"`
			} `json:"data"`
		}{}
		err := json.Unmarshal(msg, &rawDepth)
		if err != nil {
			return err
		}

		depth := this.parseDepthData(rawDepth.Data.Bids, rawDepth.Data.Asks)
		depth.Symbol = symbol
		depth.UTime = time.Now()
		this.depthCallback(depth)
		return nil
	}
	this.subscribe(endpoint, handle)
	return nil
}

func (this *binanceExchange) SubTicker(symbol string) error {
	if this.tickerCallback == nil {
		return errors.New("ticker回调函数未初始化")
	}

	endpoint := fmt.Sprintf("%s%s@miniTicker", this.combinedBaseUrl, strings.ToLower(symbol))
	//endpoint = this.combinedBaseUrl + "btcusdt@miniTicker"

	handle := func(msg []byte) error {
		//log.Info("打印消息: %v\n", string(msg))
		dataMap := make(map[string]interface{})
		err := json.Unmarshal(msg, &dataMap)
		if err != nil {
			return err
		}
		data := dataMap["data"].(map[string]interface{})
		msgType, ok := data["e"].(string)
		if !ok {
			return errors.New("消息类型错误")
		}
		switch msgType {
		case "24hrMiniTicker":

			ticker := this.parseTicker(data)
			ticker.Symbol = symbol
			this.tickerCallback(ticker)
			return nil

		default:
			return errors.New("未知消息类型")
		}
		return nil
	}
	this.subscribe(endpoint, handle)
	return nil
}

func (this *binanceExchange) SubKline(symbol string, period int) error {
	if this.klineCallback == nil {
		return errors.New("kline回调函数未初始化")
	}
	res, ok := KLINE_PERIOD[period]
	if !ok {
		res = "1m"
	}
	endpoint := fmt.Sprintf("%s%s@kline_%s", this.combinedBaseUrl, strings.ToLower(symbol), res)
	handle := func(msg []byte) error {
		dataMap := make(map[string]interface{})
		err := json.Unmarshal(msg, &dataMap)
		if err != nil {
			return err
		}

		data := dataMap["data"].(map[string]interface{})
		msgType, ok := data["e"].(string)
		if !ok {
			return errors.New("k线类型错误")
		}
		switch msgType {
		case "kline":
			k := data["k"].(map[string]interface{})
			kline := this.parseKline(k)
			kline.Symbol = symbol
			this.klineCallback(kline, period)
		default:
			return errors.New("未知数据类型")
		}
		return nil
	}
	this.subscribe(endpoint, handle)
	return nil
}

func (this *binanceExchange) parseDepthData(bids, asks [][]interface{}) *Depth {
	depth := new(Depth)
	for _, v := range bids {
		depth.BidList = append(depth.BidList, DepthRecord{ToFloat64(v[0]), ToFloat64(v[1])})
	}

	for _, v := range asks {
		depth.AskList = append(depth.AskList, DepthRecord{ToFloat64(v[0]), ToFloat64(v[1])})
	}
	return depth
}

func (this *binanceExchange) parseTicker(tickerMap map[string]interface{}) *Ticker {
	ticker := &Ticker{
		Last: ToFloat64(tickerMap["c"]),
		Buy:  ToFloat64(tickerMap["b"]),
		Sell: ToFloat64(tickerMap["a"]),
		High: ToFloat64(tickerMap["h"]),
		Low:  ToFloat64(tickerMap["l"]),
		Vol:  ToFloat64(tickerMap["v"]),
	}
	//ticker := new(Ticker)
	//ticker.Date = ToUint64(tickerMap["E"])
	//ticker.Last = ToFloat64(tickerMap["c"])
	//ticker.Vol = ToFloat64(tickerMap["v"])
	//ticker.Low = ToFloat64(tickerMap["l"])
	//ticker.High = ToFloat64(tickerMap["h"])
	//ticker.Buy = ToFloat64(tickerMap["b"])
	//ticker.Sell = ToFloat64(tickerMap["a"])
	return ticker
}

func (this *binanceExchange) parseKline(k map[string]interface{}) *Kline {
	kline := &Kline{
		Timestamp: int64(ToInt(k["t"])),
		Open:      ToFloat64(k["o"]),
		Close:     ToFloat64(k["o"]),
		High:      ToFloat64(k["o"]),
		Low:       ToFloat64(k["o"]),
		Vol:       ToFloat64(k["o"]),
	}
	return kline
}

func (this *binanceExchange) SetCallbacks(depthCallback func(*Depth), tickerCallback func(*Ticker), klineCallback func(*Kline, int)) {
	this.depthCallback = depthCallback
	this.tickerCallback = tickerCallback
	this.klineCallback = klineCallback
}
