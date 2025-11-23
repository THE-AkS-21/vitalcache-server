package kafka

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer() (*Producer, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		// If no brokers configured, return nil (optional feature)
		return nil, nil
	}
	addrs := strings.Split(brokers, ",")

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	p, err := sarama.NewSyncProducer(addrs, config)
	if err != nil {
		return nil, err
	}

	slog.Info("connected to kafka", "brokers", brokers)
	return &Producer{producer: p, topic: "vitalcache.events"}, nil
}

func (p *Producer) Publish(event string, payload interface{}) error {
	if p == nil || p.producer == nil {
		return nil
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event),
		Value: sarama.ByteEncoder(b),
	}

	_, _, err = p.producer.SendMessage(msg)
	return err
}

func (p *Producer) Close() error {
	if p != nil && p.producer != nil {
		return p.producer.Close()
	}
	return nil
}
