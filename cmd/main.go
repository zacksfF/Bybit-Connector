// cmd/main.go
package main

import (
	"bybit_connector/handler"
	"bybit_connector/internal/config"
	"bybit_connector/pkg/market"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or cannot be loaded: %v", err)
	}

	// Initialize configuration
	cfg := &config.Config{
		WebSocketURL:   getEnvWithDefault("TESTNET_MODE", "false") == "true",
			"wss://stream-testnet.bybit.com/v5/public/spot" : 
			"wss://stream.bybit.com/v5/public/spot",
		Symbol:         getEnvWithDefault("SYMBOL", "BTCUSDT"),
		APIKey:         os.Getenv("API_KEY"),
		APISecret:      os.Getenv("API_SECRET"),
		HeartbeatTimer: getDurationEnv("HEARTBEAT_INTERVAL", 20),
		ReconnectTimer: getDurationEnv("RECONNECT_INTERVAL", 5),
		OrderBookDepth: getIntEnv("ORDERBOOK_DEPTH", 25),
	}

	// Initialize order book
	orderBook := market.NewOrderBookLocal()
	
	// Create WebSocket handler
	wsHandler := handler.NewWebSocketHandler(orderBook)
	
	// Create WebSocket client
	wsClient, err := socket.NewWebSocketClient(cfg.WebSocketURL, wsHandler)
	if err != nil {
		log.Fatalf("Failed to create WebSocket client: %v", err)
	}

	// Connect to WebSocket
	if err := wsClient.Connect(); err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer wsClient.Disconnect()

	// Subscribe to orderbook
	if err := wsClient.SubscribeToOrderbook(cfg.Symbol); err != nil {
		log.Fatalf("Failed to subscribe to orderbook: %v", err)
	}

	// Monitor connection status and handle reconnection
	go monitorConnection(wsClient, cfg)

	// Setup ticker to print orderbook status periodically
	go monitorOrderbook(orderBook)

	// Wait for termination signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutting down...")
}

// monitorConnection checks connection status and reconnects if needed
func monitorConnection(wsClient *socket.WebSocketClient, cfg *config.Config) {
	ticker := time.NewTicker(cfg.ReconnectTimer * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		if !wsClient.IsConnected() {
			log.Println("WebSocket disconnected, attempting to reconnect...")
			if err := wsClient.Connect(); err != nil {
				log.Printf("Failed to reconnect: %v", err)
			} else {
				// Resubscribe on reconnection
				if err := wsClient.SubscribeToOrderbook(cfg.Symbol); err != nil {
					log.Printf("Failed to resubscribe to orderbook: %v", err)
				} else {
					log.Println("Successfully reconnected and resubscribed")
				}
			}
		}
	}
}

// monitorOrderbook periodically prints the state of the order book
func monitorOrderbook(orderBook market.OrderBookLocal) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		ob := &market.OrderBook{
			Bids: []market.Item{},
			Asks: []market.Item{},
		}
		ticker := &market.Ticker{}
		orderBook.GetOrderBook(ob, ticker)
		
		if len(ob.Bids) > 0 && len(ob.Asks) > 0 {
			fmt.Printf("[%s] Current Orderbook - Top Bid: %.2f (%.6f), Top Ask: %.2f (%.6f), Spread: %.2f\n", 
				time.Now().Format("15:04:05"),
				ticker.Bid, ticker.BidSize, 
				ticker.Ask, ticker.AskSize,
				ticker.Ask - ticker.Bid)
		} else {
			log.Println("Orderbook is empty")
		}
	}
}

// Helper functions for environment variables
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getIntEnv(key string, defaultValue int) int {
	strValue := os.Getenv(key)
	if strValue == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(strValue)
	if err != nil {
		log.Printf("Warning: Invalid value for %s, using default: %v", key, err)
		return defaultValue
	}
	
	return value
}

func getDurationEnv(key string, defaultValue int) time.Duration {
	return time.Duration(getIntEnv(key, defaultValue)) * time.Second
}