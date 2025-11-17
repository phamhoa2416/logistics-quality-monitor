package ingestion

import (
	"errors"
	"fmt"
	"log"
	"sync"

	pkgmqtt "logistics-quality-monitor/pkg/mqtt"
)

// MQTTIngestionConfig describes the topics and MQTT connection parameters.
type MQTTIngestionConfig struct {
	ClientConfig   *pkgmqtt.Config
	SensorTopic    string
	LocationTopic  string
	StatusTopic    string
	HeartbeatTopic string
	QoS            byte
}

// MQTTIngestionClient wires MQTT messages into the ingestion processor.
type MQTTIngestionClient struct {
	cfg       *MQTTIngestionConfig
	client    *pkgmqtt.Client
	processor *Processor

	mu            sync.Mutex
	started       bool
	subscriptions []string
}

// NewMQTTIngestionClient builds a new MQTT client for ingestion.
func NewMQTTIngestionClient(cfg *MQTTIngestionConfig, processor *Processor) (*MQTTIngestionClient, error) {
	if cfg == nil || cfg.ClientConfig == nil {
		return nil, errors.New("mqtt ingestion config is not configured")
	}
	if processor == nil {
		return nil, errors.New("processor is required")
	}

	client := pkgmqtt.NewClient(cfg.ClientConfig)
	return &MQTTIngestionClient{
		cfg:       cfg,
		client:    client,
		processor: processor,
	}, nil
}

// Start establishes the MQTT connection and subscribes to the topics.
func (c *MQTTIngestionClient) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	if err := c.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	type subscription struct {
		topic   string
		handler pkgmqtt.MessageHandler
	}

	subs := []subscription{}
	if c.cfg.SensorTopic != "" {
		subs = append(subs, subscription{
			topic:   c.cfg.SensorTopic,
			handler: c.handleSensorMessage,
		})
	}
	if c.cfg.LocationTopic != "" {
		subs = append(subs, subscription{
			topic:   c.cfg.LocationTopic,
			handler: c.handleLocationMessage,
		})
	}
	if c.cfg.StatusTopic != "" {
		subs = append(subs, subscription{
			topic:   c.cfg.StatusTopic,
			handler: c.handleStatusMessage,
		})
	}
	if c.cfg.HeartbeatTopic != "" {
		subs = append(subs, subscription{
			topic:   c.cfg.HeartbeatTopic,
			handler: c.handleHeartbeatMessage,
		})
	}

	if len(subs) == 0 {
		return errors.New("no MQTT topics configured for ingestion")
	}

	qos := c.cfg.QoS
	for _, sub := range subs {
		if err := c.client.Subscribe(sub.topic, qos, sub.handler); err != nil {
			c.client.Disconnect()
			return fmt.Errorf("subscribe failed for topic %s: %w", sub.topic, err)
		}
		c.subscriptions = append(c.subscriptions, sub.topic)
		log.Printf("📡 Listening for MQTT messages on %s", sub.topic)
	}

	c.started = true
	return nil
}

// Stop unsubscribes and disconnects from the broker.
func (c *MQTTIngestionClient) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}

	if len(c.subscriptions) > 0 {
		if err := c.client.Unsubscribe(c.subscriptions...); err != nil {
			log.Printf("failed to unsubscribe from MQTT topics: %v", err)
		}
	}

	c.client.Disconnect()
	c.started = false
	c.subscriptions = nil
}

// handleSensorMessage decodes sensor data, validates it and hands it to the processor.
func (c *MQTTIngestionClient) handleSensorMessage(_ string, payload []byte) {
	msg, err := ParseSensorData(payload)
	if err != nil {
		log.Printf("invalid sensor payload: %v", err)
		return
	}

	c.processor.ProcessSensorData(msg)
}

// handleLocationMessage decodes GPS data and hands it to the processor.
func (c *MQTTIngestionClient) handleLocationMessage(_ string, payload []byte) {
	msg, err := ParseLocationData(payload)
	if err != nil {
		log.Printf("invalid location payload: %v", err)
		return
	}

	c.processor.ProcessLocationData(msg)
}

// handleStatusMessage currently logs status updates (placeholder for future handling).
func (c *MQTTIngestionClient) handleStatusMessage(topic string, payload []byte) {
	log.Printf("status update received on %s: %s", topic, string(payload))
}

// handleHeartbeatMessage currently logs heartbeat updates (placeholder for future handling).
func (c *MQTTIngestionClient) handleHeartbeatMessage(topic string, payload []byte) {
	log.Printf("heartbeat message received on %s: %s", topic, string(payload))
}
