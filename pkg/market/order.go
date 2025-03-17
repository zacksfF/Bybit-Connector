package market

import (
	"strconv"
	"time"
)

type Order struct {
	OrderID        string    `json:"order_id"`
	OrderLinkID    string    `json:"order_link_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	OrderType      string    `json:"order_type"`
	Price          float32   `json:"price"`
	Qty            float64   `json:"qty"`
	TimeInForce    string    `json:"time_in_force"`
	CreateType     string    `json:"create_type"`
	CancelType     string    `json:"cancel_type"`
	OrderStatus    string    `json:"order_status"`
	LeavesQty      float64   `json:"leaves_qty"`
	CumExecQty     float64   `json:"cum_exec_qty"`
	CumExecValue   float64   `json:"cum_exec_value,string"`
	CumExecFee     float64   `json:"cum_exec_fee,string"`
	Timestamp      time.Time `json:"timestamp"`
	TakeProfit     float64   `json:"take_profit,string"`
	StopLoss       float64   `json:"stop_loss,string"`
	TrailingStop   float64   `json:"trailing_stop,string"`
	TrailingActive float64   `json:"trailing_active,string"`
	LastExecPrice  float64   `json:"last_exec_price,string"`
	ReduceOnly     bool      `json:"reduce_only,bool"`
	CloseOnTrigger bool      `json:"close_on_trigger,bool"`
}

// OrderBook represents the order book for a specific symbol.
type OrderBook struct {
	Bids      []Item    `json:"bids"`
	Asks      []Item    `json:"asks"`
	Timestamp time.Time `json:"timestamp"`
}

// Item trandformed data types
type Item struct {
	Amount float64 `json:"amount"`
	Price  float64 `json:"price"`
}

// Recevied data types from exchange
type OrderBookL2 struct {
	ID     int64   `json:"id"`
	Price  float64 `json:"price"`
	Side   string  `json:"side"`
	Size   float64 `json:"size"`
	Symbol string  `json:"symbol"`
}

type OrderBookL2Delta struct {
	Delete []*OrderBookL2 `json:"delete"`
	Update []*OrderBookL2 `json:"update"`
	Insert []*OrderBookL2 `json:"insert"`
}

// calling o.Bids and o.Asks
func (o *OrderBook) GetBids() []Item {
	return o.Bids
}

func (o *OrderBookL2) Key() string {
	return strconv.FormatInt(o.ID, 10)
}

