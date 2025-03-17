package market

import "time"

type Trade struct {
	Time time.Time `json:"time"`
	TradeTimeMs string `json:"trade_time_ms"`
	Symbol string `json:"symbol"`
	Side string `json:"side"`
	Size float64 `json:"size"`
	Price float64 `json:"price"`
	TickDirection string `json:"tick_direction"`
	TradeId string `json:"trade_id"`
	CrossSeq string `json:"cross_seq"`
	IsBlockTrade bool `json:"is_block_trade"`
}
