package main

import (
	"wisp/common"
	"wisp/exchange"
	"wisp/log"
)

func main() {

	var channel = make(chan struct{})
	log.InitLog()
	log.Info(common.Logo)

	binance := exchange.NewBinanceExchange()
	binance.OpenDump()
	binance.SetCallbacks(depthCallback, tickerCallback, klineCallback)
	//binance.SubDepths("btcusdt", 5)
	//binance.SubDepths("eosusdt", 5)
	//binance.SubTicker("btcusdt")
	binance.SubKline("btcusdt", common.KLINE_PERIOD_1MIN)

	<-channel
}

func depthCallback(depth *common.Depth) {
	log.Info("币安 深度数据: %s %v\n", depth.Symbol, depth.BidList)
}

func tickerCallback(ticker *common.Ticker) {
	log.Info("币安 成交数据: %s %v\n", ticker.Symbol, ticker.Last)
}

func klineCallback(kline *common.Kline, period int) {
	log.Info("币安 K线数据: %s %v\n", kline.Symbol, kline.Timestamp)
}
