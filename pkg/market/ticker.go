package market

import "time"

//
type Ticker struct {
	Bid     float64   `json:"bid"`
	BidSize float64   `json:"bid_size"`
	Ask     float64   `json:"ask"`
	AskSize float64   `json:"ask_size"`
	Time    time.Time `json:"time"`
}
