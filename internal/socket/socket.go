package socket

import (
	"bybit_connector/internal/config"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket connection implementation
type WebSocketClient struct {
	Conn            *websocket.Conn
	URL             string
	APIKey          string
	APISecret       string
	Subscription    map[string]bool
	MessageHandler  func([]byte)
	ErrorHandler    func(error)
	Done            chan struct{}
	Config          *config.Config
	IsAuthenticated bool
}

// Message represents the basic structure of Bybit WebSocket messages
type Message struct {
	Topic string      `json:"topic,omitempty"`
	Type  string      `json:"type,omitempty"`
	Data  interface{} `json:"omitempty"`
	Ts    int64       `json:"ts,omitempty"`
	Op    string      `json:"op,omitempty"`
	Args  []string    `json:"args,omitempty"`
}

// AuthMessage reprsents an authentication message for private channels
type AuthMessage struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// NewWebSocketClient craetes a new Bybit webSocket client
func NewWebSocketClient(config *config.Config, messageHandler func([]byte), errorHandler func(error)) *WebSocketClient {
	return &WebSocketClient{
		URL:             config.BybitWSBaseURL,
		APIKey:          config.BybitAPIKey,
		APISecret:       config.BybitAPISecret,
		Subscription:    make(map[string]bool),
		MessageHandler:  messageHandler,
		ErrorHandler:    errorHandler,
		Done:            make(chan struct{}),
		Config:          config,
		IsAuthenticated: false,
	}
}

/*
Connect()
Authenticate()
Subscribe()
Unsubscrbe()
Close()
listen()
tyrReconnect()
KeepAlive()
SendJson()
*/

// connect esatablihses a webSocket connection to Bybit
func (c *WebSocketClient) Connect() error {
	var err error
	c.Conn, _, err = websocket.DefaultDialer.Dial(c.URL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial error: %w", err)
	}

	//Start listeners
	go c.listen() // we wil define it later
	go c.keepAlive()

	//Athenticate if credential are provided
	if c.APIKey != "" && c.APISecret != "" {
		err = c.authenticate()
		if err != nil {
			return fmt.Errorf("authentication error: %w", err)
		}
	}
	return nil
}

// authenticate sends authentication message for private channels
func (c *WebSocketClient) authenticate() error {
	//generate timestamp (expiry)
	expires := time.Now().Unix()*1000 + 10000 // 10 seconds from now

	// create signature
	signaturePayload := fmt.Sprintf("GET/realtime%d", expires)
	h := hmac.New(sha256.New, []byte(c.APISecret))
	h.Write([]byte(signaturePayload))
	signature := hex.EncodeToString(h.Sum(nil))

	//create autyh message
	auth := AuthMessage{
		Op: "auth",
		Args: []string{
			c.APIKey,
			fmt.Sprintf("%d", expires),
			signature,
		},
	}

	//send auth message
	err := c.sendJSON(auth)
	if err != nil {
		return err
	}

	c.IsAuthenticated = true
	return nil
}

// Subscibe subscribes to one or more topicvs
func (c *WebSocketClient) Subscibe(topics []string) error {
	request := Message{
		Op:   "subscribe",
		Args: topics,
	}

	for _, topic := range topics {
		c.Subscription[topic] = true
	}

	return c.sendJSON(request)
}

// Unsubscribe uns from onr or morw topics
func (c *WebSocketClient) Unsubscribe(topics []string) error {
	request := Message{
		Op:   "unsubscribe",
		Args: topics,
	}

	for _, topic := range topics {
		delete(c.Subscription, topic)
	}

	return c.sendJSON(request)
}

// Close closes the webSocket connection
func (c *WebSocketClient) close() {
	close(c.Done)
	if c.Conn != nil {
		c.Conn.Close()
	}
}

// listen continuously reads message fron the websocket
func (c *WebSocketClient) listen() {
	defer func() {
		if c.Conn != nil {
			c.Conn.Close()
		}
	}()

	for {
		select {
		case <-c.Done:
			return
		default:
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				if c.ErrorHandler != nil {
					c.ErrorHandler(fmt.Errorf("read error: %w", err))
				}
				//try to reconnect
				c.tyrReconnect()
				return
			}

			//handle pong message
			if string(message) == "pong" {
				continue
			}

			if c.MessageHandler != nil {
				c.MessageHandler(message)
			}
		}
	}
}

// tryreconnect attme
func (c *WebSocketClient) tyrReconnect() {
	log.Printf("COnnection lost, Attempting to reconnect...")

	//Wait before reconnecting
	time.Sleep(time.Duration(c.Config.PingInterval) * time.Second)

	//save current subscribtions
	subscriptions := []string{}
	for topic := range c.Subscription {
		subscriptions = append(subscriptions, topic)
	}

	//Close current connection if it exists
	if c.Conn != nil {
		c.Conn.Close()
	}

	//Connect again
	err := c.Connect()
	if err != nil {
		if c.ErrorHandler != nil {
			c.ErrorHandler(fmt.Errorf("reconnection error: %w", err))
		}
		return
	}

	//resubscrbe to topics
	if len(subscriptions) > 0 {
		err = c.Subscibe(subscriptions)
		if err != nil {
			c.ErrorHandler(fmt.Errorf("resubsciption error: %w", err))
		}
	}
}

// KeepAlive sends periodic pings to keep the connection alive
func (c *WebSocketClient) keepAlive() {
	ticker := time.NewTicker(time.Duration(c.Config.PingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.Conn != nil {
				if err := c.Conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
					if c.ErrorHandler != nil {
						c.ErrorHandler(fmt.Errorf("ping error: %w", err))
					}

					//try to reconnect
					c.tyrReconnect()
					return
				}
			}
		case <-c.Done:
			return
		}
	}
}

// sendJSON sends a JSON message trough teh websocket
func (c *WebSocketClient) sendJSON(v interface{}) error {
	if c.Conn == nil {
		return fmt.Errorf("connecting is nil")
	}
	return c.Conn.WriteJSON(v)
}
