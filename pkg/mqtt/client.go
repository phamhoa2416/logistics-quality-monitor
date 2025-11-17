package mqtt

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

type Config struct {
	Broker               string
	ClientID             string
	Username             string
	Password             string
	CleanSession         bool
	KeepAlive            int
	ConnectTimeout       int
	AutoReconnect        bool
	MaxReconnectInterval time.Duration
}

type Client struct {
	client mqtt.Client
	config *Config
}

type MessageHandler func(topic string, payload []byte)

func NewClient(config *Config) *Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetCleanSession(config.CleanSession)
	opts.SetKeepAlive(time.Duration(config.KeepAlive) * time.Second)
	opts.SetConnectTimeout(time.Duration(config.ConnectTimeout) * time.Second)
	opts.SetAutoReconnect(config.AutoReconnect)
	opts.SetMaxReconnectInterval(config.MaxReconnectInterval)

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("mqtt client connected")
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("mqtt connection lost: %v", err)
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		log.Println("reconnecting to MQTT broker...")
	})

	client := mqtt.NewClient(opts)

	return &Client{
		client: client,
		config: config,
	}
}

// Connect establishes a connection to the MQTT broker
func (c *Client) Connect() error {
	log.Printf("Connecting to MQTT broker at %s", c.config.Broker)

	token := c.client.Connect()
	token.Wait()

	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	return nil
}

// Subscribe subscribes to a topic with handler
func (c *Client) Subscribe(topic string, qos byte, handler MessageHandler) error {
	log.Printf("Subscribing to topic: %s (QoS: %d)", topic, qos)

	token := c.client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	})

	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	log.Printf("Subscribed to topic: %s", topic)
	return nil
}

// Publish publishes a message to a topic
func (c *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	token := c.client.Publish(topic, qos, retained, payload)
	token.Wait()
	return token.Error()
}

// Unsubscribe unsubscribes from a topic
func (c *Client) Unsubscribe(topics ...string) error {
	token := c.client.Unsubscribe(topics...)
	token.Wait()
	return token.Error()
}

// Disconnect disconnects from MQTT broker
func (c *Client) Disconnect() {
	log.Println("Disconnecting from MQTT broker...")
	c.client.Disconnect(250)
	log.Println("Disconnected from MQTT broker")
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}
