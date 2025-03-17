package socket

import (
	"bybit_connector/internal/config"
	"log"
	"testing"
	"time"
)

func TestWebSocketClient(t *testing.T) {
	// Load config
	config, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create message handler
	messageHandler := func(message []byte) {
		log.Printf("Received message: %s", string(message))
	}

	// Create error handler
	errorHandler := func(err error) {
		log.Printf("Error: %v", err)
	}

	// Create WebSocket client
	client := NewWebSocketClient(config, messageHandler, errorHandler)

	// Connect
	err = client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	//defer client.Close()
	defer client.close()

	// Subscribe to orderbook
	err = client.Subscibe([]string{"orderbook.1.BTCUSDT"})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Wait for some messages
	time.Sleep(5 * time.Second)

	// Unsubscribe
	err = client.Unsubscribe([]string{"orderbook.1.BTCUSDT"})
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Subscribe to trade
	err = client.Subscibe([]string{"trade.BTCUSDT"})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Wait for some messages
	time.Sleep(5 * time.Second)
}
