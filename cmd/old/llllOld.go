package main

//Entry point of the application

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// BybitClient represents the WebSocket client
type BybitClient struct {
	conn           *websocket.Conn
	url            string
	subscriptions  map[string]bool
	messageHandler func([]byte)
	done           chan struct{}
}

// Message represents the basic structure of Bybit WebSocket messages
type Message struct {
	Topic string      `json:"topic"`
	Type  string      `json:"type"`
	Data  interface{} `json:"data"`
	Ts    int64       `json:"ts"`
}

// SubscriptionRequest represents a subscription request
type SubscriptionRequest struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// NewBybitClient creates a new Bybit WebSocket client
func NewBybitClient(url string, messageHandler func([]byte)) *BybitClient {
	return &BybitClient{
		url:            url,
		subscriptions:  make(map[string]bool),
		messageHandler: messageHandler,
		done:           make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to Bybit
func (c *BybitClient) Connect() error {
	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return err
	}

	// Start listening for messages
	go c.listen()

	// Start ping/pong to keep connection alive
	go c.keepAlive()

	return nil
}

// Subscribe subscribes to one or more topics
func (c *BybitClient) Subscribe(topics []string) error {
	request := SubscriptionRequest{
		Op:   "subscribe",
		Args: topics,
	}

	for _, topic := range topics {
		c.subscriptions[topic] = true
	}

	return c.sendJSON(request)
}

// Unsubscribe unsubscribes from one or more topics
func (c *BybitClient) Unsubscribe(topics []string) error {
	request := SubscriptionRequest{
		Op:   "unsubscribe",
		Args: topics,
	}

	for _, topic := range topics {
		delete(c.subscriptions, topic)
	}

	return c.sendJSON(request)
}

// Close closes the WebSocket connection
func (c *BybitClient) Close() {
	close(c.done)
	c.conn.Close()
}

// listen continuously reads messages from the WebSocket
func (c *BybitClient) listen() {
	defer c.conn.Close()

	for {
		select {
		case <-c.done:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				return
			}

			if c.messageHandler != nil {
				c.messageHandler(message)
			}
		}
	}
}

// keepAlive sends periodic pings to keep the connection alive
func (c *BybitClient) keepAlive() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				log.Println("ping error:", err)
				return
			}
		case <-c.done:
			return
		}
	}
}

// sendJSON sends a JSON message through the WebSocket
func (c *BybitClient) sendJSON(v interface{}) error {
	return c.conn.WriteJSON(v)
}

func main() {
	// Example usage
	client := NewBybitClient("wss://stream.bybit.com/v5/public/spot", func(msg []byte) {
		log.Printf("Received: %s", string(msg))
	})

	if err := client.Connect(); err != nil {
		log.Fatal("Connection error:", err)
	}

	// Subscribe to some topics
	if err := client.Subscribe([]string{"orderbook.1.BTCUSDT"}); err != nil {
		log.Fatal("Subscription error:", err)
	}

	// Wait for interrupt signal to gracefully close the connection
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	<-interrupt
	log.Println("Closing connection...")
	client.Close()
}
