// internal/parser.go
package parser

import (
	"bybit_connector/pkg/market"
	"encoding/json"
	"fmt"
	"time"
)

// MessageParser handles the parsing of different types of WebSocket messages
type MessageParser struct {
	OrderBookLocal *market.OderBookLocal
}

// NewMessageParser creates a new message parser
func NewMessageParser() *MessageParser {
	return &MessageParser{
		OrderBookLocal: market.NewOrderBookLocal(),
	}
}

// ParseMessage parses a WebSocket message into the appropriate type
func (p *MessageParser) ParseMessage(message []byte) (interface{}, error) {
	// First determine the type of message
	var baseMsg struct {
		Topic   string      `json:"topic,omitempty"`
		Type    string      `json:"type,omitempty"`
		Data    interface{} `json:"data,omitempty"`
		Ts      int64       `json:"ts,omitempty"`
		Op      string      `json:"op,omitempty"`
		Ret_msg string      `json:"ret_msg,omitempty"`
		Success bool        `json:"success,omitempty"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base message: %w", err)
	}

	// If it's a response to a subscription request
	if baseMsg.Success && baseMsg.Ret_msg == "subscribe" {
		return baseMsg, nil
	}

	// Process based on topic if available
	if baseMsg.Topic != "" {
		// Extract the topic type (e.g., "orderbook" from "orderbook.1.BTCUSDT")
		topicParts := splitTopic(baseMsg.Topic)
		if len(topicParts) < 3 {
			return nil, fmt.Errorf("invalid topic format: %s", baseMsg.Topic)
		}

		topicType := topicParts[0]
		symbol := topicParts[2]

		switch topicType {
		case "orderbook":
			return p.parseOrderbook(message, baseMsg.Type, symbol)
		case "trade":
			return p.parseTrade(message, symbol)
		case "ticker":
			return p.parseTicker(message, symbol)
		default:
			return baseMsg, nil
		}
	}

	// Default: return the base message
	return baseMsg, nil
}

// parseOrderbook parses orderbook messages
func (p *MessageParser) parseOrderbook(message []byte, msgType, symbol string) (interface{}, error) {
	var orderbookMsg struct {
		Topic string `json:"topic"`
		Type  string `json:"type"`
		Ts    int64  `json:"ts"`
		Data  struct {
			S   string     `json:"s"`   // Symbol
			B   [][]string `json:"b"`   // Bids
			A   [][]string `json:"a"`   // Asks
			U   int64      `json:"u"`   // Update ID
			Seq int64      `json:"seq"` // Sequence
		} `json:"data"`
		Cts int64 `json:"cts"`
	}

	if err := json.Unmarshal(message, &orderbookMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orderbook message: %w", err)
	}

	// Handle snapshot or delta based on the message type
	if msgType == "snapshot" {
		// Create a new snapshot
		snapshot := []*market.OrderBookL2{}

		// Process bids
		for i, bid := range orderbookMsg.Data.B {
			if len(bid) < 2 {
				continue
			}
			price, err := parseFloat(bid[0])
			if err != nil {
				continue
			}
			size, err := parseFloat(bid[1])
			if err != nil {
				continue
			}
			snapshot = append(snapshot, &market.OrderBookL2{
				ID:     int64(i + 1), // Using index as ID for simplicity
				Price:  price,
				Side:   "Buy",
				Size:   size,
				Symbol: symbol,
			})
		}

		// Process asks
		for i, ask := range orderbookMsg.Data.A {
			if len(ask) < 2 {
				continue
			}
			price, err := parseFloat(ask[0])
			if err != nil {
				continue
			}
			size, err := parseFloat(ask[1])
			if err != nil {
				continue
			}
			snapshot = append(snapshot, &market.OrderBookL2{
				ID:     int64(i + 1000), // Offset for asks
				Price:  price,
				Side:   "Sell",
				Size:   size,
				Symbol: symbol,
			})
		}

		// Load the snapshot into the local orderbook
		err := p.OrderBookLocal.LoadSnapshot(snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to load orderbook snapshot: %w", err)
		}

		return snapshot, nil
	} else if msgType == "delta" {
		// Handle delta update
		delta := &market.OrderBookL2Delta{
			Delete: []*market.OrderBookL2{},
			Update: []*market.OrderBookL2{},
			Insert: []*market.OrderBookL2{},
		}

		// Process bid updates (treating all as updates for simplicity)
		for i, bid := range orderbookMsg.Data.B {
			if len(bid) < 2 {
				continue
			}
			price, err := parseFloat(bid[0])
			if err != nil {
				continue
			}
			size, err := parseFloat(bid[1])
			if err != nil {
				continue
			}
			delta.Update = append(delta.Update, &market.OrderBookL2{
				ID:     int64(i + 1),
				Price:  price,
				Side:   "Buy",
				Size:   size,
				Symbol: symbol,
			})
		}

		// Process ask updates
		for i, ask := range orderbookMsg.Data.A {
			if len(ask) < 2 {
				continue
			}
			price, err := parseFloat(ask[0])
			if err != nil {
				continue
			}
			size, err := parseFloat(ask[1])
			if err != nil {
				continue
			}
			delta.Update = append(delta.Update, &market.OrderBookL2{
				ID:     int64(i + 1000),
				Price:  price,
				Side:   "Sell",
				Size:   size,
				Symbol: symbol,
			})
		}

		// Update the local orderbook
		p.OrderBookLocal.Update(delta)

		return delta, nil
	}

	return orderbookMsg, nil
}

// parseTrade parses trade messages
func (p *MessageParser) parseTrade(message []byte, symbol string) (*market.Trade, error) {
	var tradeMsg struct {
		Topic string `json:"topic"`
		Data  []struct {
			Timestamp     string `json:"T"`
			Symbol        string `json:"s"`
			Side          string `json:"S"`
			Size          string `json:"v"`
			Price         string `json:"p"`
			TickDirection string `json:"L"`
			TradeId       string `json:"i"`
			IsBlockTrade  bool   `json:"BT"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &tradeMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trade message: %w", err)
	}

	if len(tradeMsg.Data) == 0 {
		return nil, fmt.Errorf("no trade data found")
	}

	// Convert the first trade
	data := tradeMsg.Data[0]
	t, err := parseTimestamp(data.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trade timestamp: %w", err)
	}

	size, err := parseFloat(data.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trade size: %w", err)
	}

	price, err := parseFloat(data.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trade price: %w", err)
	}

	return &market.Trade{
		Time:          t,
		TradeTimeMs:   data.Timestamp,
		Symbol:        data.Symbol,
		Side:          data.Side,
		Size:          size,
		Price:         price,
		TickDirection: data.TickDirection,
		TradeId:       data.TradeId,
		IsBlockTrade:  data.IsBlockTrade,
	}, nil
}

// parseTicker parses ticker messages
func (p *MessageParser) parseTicker(message []byte, symbol string) (*market.Ticker, error) {
	var tickerMsg struct {
		Topic string `json:"topic"`
		Data  struct {
			Symbol    string `json:"symbol"`
			LastPrice string `json:"lastPrice"`
			BidPrice  string `json:"bidPrice"`
			BidSize   string `json:"bidSize"`
			AskPrice  string `json:"askPrice"`
			AskSize   string `json:"askSize"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}

	if err := json.Unmarshal(message, &tickerMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ticker message: %w", err)
	}

	bidPrice, err := parseFloat(tickerMsg.Data.BidPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bid price: %w", err)
	}

	bidSize, err := parseFloat(tickerMsg.Data.BidSize)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bid size: %w", err)
	}

	askPrice, err := parseFloat(tickerMsg.Data.AskPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ask price: %w", err)
	}

	askSize, err := parseFloat(tickerMsg.Data.AskSize)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ask size: %w", err)
	}

	timestamp, err := parseTimestamp(tickerMsg.Data.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ticker timestamp: %w", err)
	}

	return &market.Ticker{
		Bid:     bidPrice,
		BidSize: bidSize,
		Ask:     askPrice,
		AskSize: askSize,
		Time:    timestamp,
	}, nil
}

// Helper function to split a topic string
func splitTopic(topic string) []string {
	var result []string
	var currentPart string
	inDots := false

	for _, char := range topic {
		if char == '.' {
			if inDots {
				currentPart += string(char)
			} else {
				if currentPart != "" {
					result = append(result, currentPart)
					currentPart = ""
				}
				inDots = true
			}
		} else {
			currentPart += string(char)
			inDots = false
		}
	}

	if currentPart != "" {
		result = append(result, currentPart)
	}

	return result
}

// Helper function to parse a float from string
func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// Helper function to parse a timestamp
func parseTimestamp(s string) (time.Time, error) {
	// Try parsing as Unix milliseconds
	var ms int64
	_, err := fmt.Sscanf(s, "%d", &ms)
	if err == nil {
		return time.Unix(0, ms*int64(time.Millisecond)), nil
	}

	// Try standard time format
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}
