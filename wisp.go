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
	binance.SubDepths("btcusdt", 5)
	binance.SubDepths("ethusdt", 5)
	binance.SubDepths("ltcusdt", 5)
	binance.SubDepths("etcusdt", 5)
	binance.SubDepths("bchusdt", 5)
	binance.SubDepths("dashusdt", 5)
	binance.SubDepths("eosusdt", 5)
	binance.SubDepths("xrpusdt", 5)
	binance.SubDepths("adausdt", 5)

	binance.SubTicker("btcusdt")
	binance.SubTicker("ethusdt")
	binance.SubTicker("ltcusdt")
	binance.SubTicker("etcusdt")
	binance.SubTicker("bchusdt")
	binance.SubTicker("dashusdt")
	binance.SubTicker("eosusdt")
	binance.SubTicker("xrpusdt")
	binance.SubTicker("adausdt")

	binance.SubKline("btcusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("ethusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("ltcusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("etcusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("bchusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("dashusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("eosusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("xrpusdt", common.KLINE_PERIOD_1MIN)
	binance.SubKline("adausdt", common.KLINE_PERIOD_1MIN)

	<-channel
}

func depthCallback(depth *common.Depth) {
	log.Info("币安 交易标的: %s 买5档: %v   卖5档: %v \n", depth.Symbol, depth.BidList, depth.AskList)
}

func tickerCallback(ticker *common.Ticker) {
	log.Info("币安 交易标的: %s 最新价: %f 最高价: %f 成交量: %f \n", ticker.Symbol, ticker.Last, ticker.High, ticker.Vol)
}

func klineCallback(kline *common.Kline, period int) {
	log.Info("币安 交易标的: %s  K线类型: %d 开盘价: %f 收盘价: %f 最高价: %f 最低价: %f \n", kline.Symbol, period, kline.Open, kline.Close, kline.High, kline.Low)
}
