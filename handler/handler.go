package handler

import (
	"bybit_connector/internal/parser"
	"bybit_connector/pkg/market"
	"encoding/json"
	"log"
)

// HTTP/WebSocket handlers
// WebSocketHandler handles WebSocket messages
type WebSocketHandler struct {
	Parser        *parser.MessageParser
	OrderBookMap  map[string]*market.OrderBook
	TickerMap     map[string]*market.Ticker
	TradeMap      map[string][]*market.Trade
	MaxTradeCount int
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		Parser:        parser.NewMessageParser(),
		OrderBookMap:  make(map[string]*market.OrderBook),
		TickerMap:     make(map[string]*market.Ticker),
		TradeMap:      make(map[string][]*market.Trade),
		MaxTradeCount: 100, // Keep the last 100 trades
	}
}

// HandleMessage processes incoming WebSocket messages
func (h *WebSocketHandler) HandleMessage(message []byte) {
	if len(message) == 0 {
		log.Println("Received empty message")
		return
	}

	// Parse the message
	parsedMsg, err := h.Parser.ParseMessage(message)
	if err != nil {
		log.Printf("Error parsing message: %v", err)
		return
	}

	// Process the message based on type
	switch msg := parsedMsg.(type) {
	case *market.OrderBookL2Delta:
		if msg != nil {
			h.handleOrderBookUpdate()
		}
	case []*market.OrderBookL2:
		if len(msg) > 0 {
			h.handleOrderBookUpdate()
		}
	case *market.Trade:
		if msg != nil {
			h.addTrade(msg)
		}
	case *market.Ticker:
		if msg != nil && msg.Symbol != "" {
			h.TickerMap[msg.Symbol] = msg
		}
	default:
		h.handleDefaultMessage(message)
	}
}

// handleOrderBookUpdate updates all orderbooks
func (h *WebSocketHandler) handleOrderBookUpdate() {
	for symbol := range h.OrderBookMap {
		h.updateOrderBook(symbol)
	}
}

// handleDefaultMessage processes subscription and other messages
func (h *WebSocketHandler) handleDefaultMessage(message []byte) {
	var baseMsg struct {
		Topic   string `json:"topic,omitempty"`
		Success bool   `json:"success,omitempty"`
	}

	if err := json.Unmarshal(message, &baseMsg); err != nil {
		log.Printf("Error unmarshaling default message: %v", err)
		return
	}

	if baseMsg.Success {
		log.Printf("Successfully subscribed/unsubscribed")
		return
	}

	if baseMsg.Topic != "" {
		parts := internal.SplitTopic(baseMsg.Topic)
		if len(parts) >= 3 {
			h.ensureOrderBook(parts[2])
		}
	}
}

// GetOrderBook gets the current orderbook for a symbol
func (h *WebSocketHandler) GetOrderBook(symbol string) *market.OrderBook {
	if symbol == "" {
		return nil
	}
	h.ensureOrderBook(symbol)
	return h.OrderBookMap[symbol]
}

// GetTicker gets the current ticker for a symbol
func (h *WebSocketHandler) GetTicker(symbol string) *market.Ticker {
	if symbol == "" {
		return nil
	}
	return h.TickerMap[symbol]
}

// GetTrades gets the trade history for a symbol
func (h *WebSocketHandler) GetTrades(symbol string) []*market.Trade {
	if symbol == "" {
		return nil
	}
	return h.TradeMap[symbol]
}

// ensureOrderBook ensures that an order book exists for the given symbol
func (h *WebSocketHandler) ensureOrderBook(symbol string) {
	if _, exists := h.OrderBookMap[symbol]; !exists {
		h.OrderBookMap[symbol] = &market.OrderBook{}
	}
}

// updateOrderBook updates the order book for the given symbol
func (h *WebSocketHandler) updateOrderBook(symbol string) {
	// Implementation for updating the order book
}

// addTrade adds a trade to the trade map
func (h *WebSocketHandler) addTrade(trade *market.Trade) {
	if trade == nil || trade.Symbol == "" {
		return
	}
	h.TradeMap[trade.Symbol] = append(h.TradeMap[trade.Symbol], trade)
	if len(h.TradeMap[trade.Symbol]) > h.MaxTradeCount {
		h.TradeMap[trade.Symbol] = h.TradeMap[trade.Symbol][1:]
	}
}
