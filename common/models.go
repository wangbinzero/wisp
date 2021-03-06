package common

import "time"

type DepthRecord struct {
	Price  float64
	Amount float64
}

type DepthRecords []DepthRecord

type Depth struct {
	Symbol  string
	UTime   time.Time
	AskList DepthRecords
	BidList DepthRecords
}

type Ticker struct {
	Symbol string  `json:"omitempty"`
	Last   float64 `json:"last"`
	Buy    float64 `json:"buy"`
	Sell   float64 `json:"sell"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Vol    float64 `json:"vol"`
	Date   uint64  `json:"date"`
}

type Kline struct {
	Symbol    string  `json:"s"`
	Timestamp int64   `json:"t"`
	Open      float64 `json:"o"`
	Close     float64 `json:"c"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Vol       float64 `json:"v"`
}
