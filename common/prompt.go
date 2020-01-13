package common

import "github.com/gorilla/websocket"

var Logo = `
----------------------------
 _      ___        
| | /| / (_)__ ___ 
| |/ |/ / (_-</ _ \
|__/|__/_/___/ .__/
            /_/
Wisp 行情前置 v0.1
---------------------------
`

const (
	KLINE_PERIOD_1MIN = 1 + iota
	KLINE_PERIOD_3MIN
	KLINE_PERIOD_5MIN
	KLINE_PERIOD_15MIN
	KLINE_PERIOD_30MIN
	KLINE_PERIOD_1H
	KLINE_PERIOD_2H
	KLINE_PERIOD_4H
	KLINE_PERIOD_6H
	KLINE_PERIOD_8H
	KLINE_PERIOD_12H
	KLINE_PERIOD_1D
	KLINE_PERIOD_3D
	KLINE_PERIOD_1W
	KLINE_PERIOD_1M
)

var KLINE_PERIOD = map[int]string{
	KLINE_PERIOD_1MIN:  "1m",
	KLINE_PERIOD_3MIN:  "3m",
	KLINE_PERIOD_5MIN:  "5m",
	KLINE_PERIOD_15MIN: "15m",
	KLINE_PERIOD_30MIN: "30m",
	KLINE_PERIOD_1H:    "1h",
	KLINE_PERIOD_2H:    "2h",
	KLINE_PERIOD_4H:    "4h",
	KLINE_PERIOD_6H:    "6h",
	KLINE_PERIOD_8H:    "8h",
	KLINE_PERIOD_12H:   "12h",
	KLINE_PERIOD_1D:    "1d",
	KLINE_PERIOD_3D:    "3d",
	KLINE_PERIOD_1W:    "1w",
	KLINE_PERIOD_1M:    "1M",
}

var ChanMap = map[string]*websocket.Conn{}
