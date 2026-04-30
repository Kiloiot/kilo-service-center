// Package mqtt provides MQTT client and publisher for MIOTY message publishing.
// It implements the Publisher interface for dependency injection and testing,
// and provides topic formatting helpers for MIOTY-compliant message routing.
package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/logger"
)

// Client represents an MQTT client for KiloCenter
type Client struct {
	client        mqtt.Client
	config        *config.MQTTConfig
	logger        logger.Logger
	mu            sync.RWMutex
	subscriptions map[string]mqtt.MessageHandler
	connected     bool
}

// NewClient creates a new MQTT client
// Returns Publisher interface to enable dependency injection and testing
func NewClient(cfg *config.MQTTConfig, log logger.Logger) (Publisher, error) {
	if cfg == nil {
		return nil, errors.New(ErrMQTTConfigNil)
	}

	c := &Client{
		config:        cfg,
		logger:        log,
		subscriptions: make(map[string]mqtt.MessageHandler),
	}

	opts := mqtt.NewClientOptions()

	// Set broker URL
	brokerURL := fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port)
	if cfg.TLS.Enabled {
		brokerURL = fmt.Sprintf("ssl://%s:%d", cfg.Host, cfg.Port)
	}
	opts.AddBroker(brokerURL)

	// Set client ID
	opts.SetClientID(cfg.ClientID)

	// Set credentials if provided
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	// Configure TLS if enabled
	if cfg.TLS.Enabled {
		// Production safety guard: prevent disabling certificate verification in production
		// This ensures TLS security is maintained in production deployments
		insecureSkipVerify := cfg.TLS.InsecureSkipVerify
		if insecureSkipVerify && os.Getenv("KILOCENTER_ENV") == "production" {
			return nil, errors.New("TLS certificate verification cannot be disabled in production")
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // guarded: only allowed in non-production environments
		}

		// Load CA certificate if provided
		if cfg.TLS.CAFile != "" {
			caCert, err := os.ReadFile(cfg.TLS.CAFile)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", ErrReadCACertFailed, err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, errors.New(ErrParseCACertFailed)
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificate and key if provided
		if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
			clientCert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", ErrLoadClientCertFailed, err)
			}
			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}

		// Set server name for SNI if provided
		if cfg.TLS.ServerName != "" {
			tlsConfig.ServerName = cfg.TLS.ServerName
		}

		opts.SetTLSConfig(tlsConfig)
	}

	// Set connection options
	opts.SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)
	opts.SetPingTimeout(DefaultPingTimeout)
	opts.SetCleanSession(cfg.CleanSession)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(time.Duration(cfg.MaxReconnectInterval) * time.Second)

	// Set callbacks
	opts.SetOnConnectHandler(c.onConnect)
	opts.SetConnectionLostHandler(c.onConnectionLost)
	opts.SetReconnectingHandler(c.onReconnecting)

	// Create the client
	c.client = mqtt.NewClient(opts)

	return c, nil
}

// Connect establishes connection to the MQTT broker
func (c *Client) Connect(ctx context.Context) error {
	c.logger.InfoContext(ctx, "Connecting to MQTT broker", "host", c.config.Host, "port", c.config.Port)

	token := c.client.Connect()
	if !token.WaitTimeout(DefaultConnectTimeout) {
		return errors.New(ErrMQTTConnectionTimeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("%s: %w", ErrConnectionFailed, err)
	}

	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()

	c.logger.InfoContext(ctx, "Successfully connected to MQTT broker")
	return nil
}

// Disconnect closes the connection to the MQTT broker
func (c *Client) Disconnect(ctx context.Context) {
	c.logger.InfoContext(ctx, "Disconnecting from MQTT broker")

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	c.client.Disconnect(DefaultDisconnectQuiesce)
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.client.IsConnected()
}

// Publish publishes a message to the specified topic
func (c *Client) Publish(ctx context.Context, topic string, qos byte, retained bool, payload interface{}) error {
	if !c.IsConnected() {
		return errors.New(ErrMQTTClientNotConnected)
	}

	token := c.client.Publish(topic, qos, retained, payload)
	if !token.WaitTimeout(DefaultPublishTimeout) {
		return errors.New(ErrMQTTPublishTimeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("%s: %w", ErrPublishFailed, err)
	}

	c.logger.DebugContext(ctx, "Published message", "topic", topic, "qos", qos, "retained", retained)
	return nil
}

// Subscribe subscribes to a topic with the specified handler
// Wraps the context-aware MessageHandler to match Paho's signature
func (c *Client) Subscribe(ctx context.Context, topic string, qos byte, handler MessageHandler) error {
	if !c.IsConnected() {
		return errors.New(ErrMQTTClientNotConnected)
	}

	// Wrap our context-aware handler to match Paho's signature
	// Capture subscribe-time context for lifecycle cancellation propagation
	pahoHandler := func(_ mqtt.Client, msg mqtt.Message) {
		handler(ctx, msg.Topic(), msg.Payload())
	}

	token := c.client.Subscribe(topic, qos, pahoHandler)
	if !token.WaitTimeout(DefaultSubscribeTimeout) {
		return errors.New(ErrMQTTSubscribeTimeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("%s: %w", ErrSubscribeFailed, err)
	}

	c.mu.Lock()
	c.subscriptions[topic] = pahoHandler
	c.mu.Unlock()

	c.logger.InfoContext(ctx, "Subscribed to topic", "topic", topic, "qos", qos)
	return nil
}

// Unsubscribe unsubscribes from a topic
func (c *Client) Unsubscribe(ctx context.Context, topics ...string) error {
	if !c.IsConnected() {
		return errors.New(ErrMQTTClientNotConnected)
	}

	token := c.client.Unsubscribe(topics...)
	if !token.WaitTimeout(DefaultUnsubscribeTimeout) {
		return errors.New(ErrMQTTUnsubscribeTimeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("%s: %w", ErrUnsubscribeFailed, err)
	}

	c.mu.Lock()
	for _, topic := range topics {
		delete(c.subscriptions, topic)
	}
	c.mu.Unlock()

	c.logger.InfoContext(ctx, "Unsubscribed from topics", "topics", topics)
	return nil
}

// onConnect is called when the client connects/reconnects
// Note: Paho callbacks do not support context.Context parameters
// This is a known limitation of the Paho MQTT library
func (c *Client) onConnect(client mqtt.Client) {
	c.logger.Info("MQTT client connected")

	// Resubscribe to all topics on reconnect
	c.mu.RLock()
	subs := make(map[string]mqtt.MessageHandler)
	for topic, handler := range c.subscriptions {
		subs[topic] = handler
	}
	c.mu.RUnlock()

	for topic, handler := range subs {
		if token := client.Subscribe(topic, 1, handler); token.Wait() && token.Error() != nil {
			c.logger.Error(LogMQTTResubscribeFailed, "topic", topic, "err", token.Error())
		} else {
			c.logger.Debug("Resubscribed to topic", "topic", topic)
		}
	}

	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
}

// onConnectionLost is called when the connection is lost
func (c *Client) onConnectionLost(_ mqtt.Client, err error) {
	c.logger.Warn(LogMQTTConnectionLost, "err", err)

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
}

// onReconnecting is called when the client is reconnecting
func (c *Client) onReconnecting(_ mqtt.Client, _ *mqtt.ClientOptions) {
	c.logger.Info("MQTT client reconnecting")
}
