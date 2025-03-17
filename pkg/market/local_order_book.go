package market

import (
	"sort"
	"sync"
	"time"
)

type OderBookLocal struct {
	ob map[string]*OrderBookL2
	m  sync.Mutex
}

func (o *OderBookLocal) GetOrderBook(ob OrderBook, t Ticker) {
	for _, v := range o.ob {
		switch v.Side {
		case "Buy":
			ob.Bids = append(ob.Bids, Item{
				Price:  v.Price,
				Amount: v.Size,
			})
		case "Sell":
			ob.Asks = append(ob.Asks, Item{
				Price:  v.Price,
				Amount: v.Size,
			})
		}
	}

	sort.Slice(ob.Bids, func(i, j int) bool {
		return ob.Bids[i].Price > ob.Bids[j].Price
	})

	sort.Slice(ob.Asks, func(i, j int) bool {
		return ob.Asks[i].Price < ob.Asks[j].Price
	})

	now := time.Now()
	ob.Timestamp = now
	// o.m.Lock()
	// defer o.m.Unlock()

	t = Ticker{
		Bid:     ob.Bids[0].Price,
		BidSize: ob.Bids[0].Amount,
		Ask:     ob.Asks[0].Price,
		AskSize: ob.Asks[0].Amount,
		Time:    now,
	}

	return
}

func NewOrderBookLocal() *OderBookLocal {
	return &OderBookLocal{
		ob: make(map[string]*OrderBookL2),
	}
}

// func NewOrderBookLocal() *OrderBookLocal {
// 	o := &OrderBookLocal{
// 		ob: make(map[string]*OrderBookL2),
// 	}
// 	return o
// }

func (o *OderBookLocal) LoadSnapshot(newOrderBook []*OrderBookL2) error {
	o.m.Lock()
	defer o.m.Unlock()

	o.ob = make(map[string]*OrderBookL2)

	for _, v := range newOrderBook {
		o.ob[v.Key()] = v
	}
	return nil
}

func (o *OderBookLocal) Update(delta *OrderBookL2Delta) {
	o.m.Lock()
	defer o.m.Unlock()

	for _, e := range delta.Delete {
		delete(o.ob, e.Key())
	}

	for _, e := range delta.Update {
		if v, ok := o.ob[e.Key()]; ok {
			//price is same while id is same
			// v.price = e.price
			v.Size = e.Size
			v.Side = e.Side
		}
	}

	for _, e := range delta.Insert {
		o.ob[e.Key()] = e
	}
}
