package mqtt

import (
	"fmt"
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Subscriber struct {
	client mqtt.Client
	onMsg  func(topic string, payload []byte)
}

func NewSubscriber(url, username, password string, onMsg func(topic string, payload []byte)) (*Subscriber, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetClientID("infra-backend")
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		slog.Info("mqtt connected", "broker", url, "username", username)
	})
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		slog.Error("mqtt connection lost", "error", err)
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		return nil, fmt.Errorf("mqtt connect: %w", token.Error())
	}

	return &Subscriber{
		client: client,
		onMsg:  onMsg,
	}, nil
}

func (s *Subscriber) Subscribe(topic string, qos byte) error {
	token := s.client.Subscribe(topic, qos, func(c mqtt.Client, msg mqtt.Message) {
		slog.Debug("mqtt message received", "topic", msg.Topic(), "size", len(msg.Payload()))
		s.onMsg(msg.Topic(), msg.Payload())
	})
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("mqtt subscribe %s: %w", topic, token.Error())
	}
	slog.Info("mqtt subscribed", "topic", topic, "qos", qos)
	return nil
}

func (s *Subscriber) Close() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250)
	}
}
