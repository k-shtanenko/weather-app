package producer

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/IBM/sarama"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/models"
)

type Producer interface {
	Produce(ctx context.Context, weather models.WeatherEntity) error
	ProduceBatch(ctx context.Context, weathers []models.WeatherEntity) error
	HealthCheck(ctx context.Context) error
	Close() error
}

type ProducerFactory interface {
	CreateProducer(broker string, topic string, requiredAcks int16, maxRetries int) (Producer, error)
}

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
	logger   logger.Logger
}

type KafkaProducerFactory struct {
	logger logger.Logger
}

func NewKafkaProducerFactory() *KafkaProducerFactory {
	return &KafkaProducerFactory{logger: logger.New("info", "development").WithField("component", "kafka_producer_factory")}
}

func (f *KafkaProducerFactory) CreateProducer(
	broker string,
	topic string,
	requiredAcks int16,
	maxRetries int,
) (Producer, error) {

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.RequiredAcks(requiredAcks)
	config.Producer.Retry.Max = maxRetries
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second

	producer, err := sarama.NewSyncProducer([]string{broker}, config)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		producer: producer,
		topic:    topic,
		logger:   f.logger,
	}, nil
}

func (k *KafkaProducer) Produce(
	ctx context.Context,
	weather models.WeatherEntity,
) error {

	data, err := json.Marshal(weather)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: k.topic,
		Value: sarama.ByteEncoder(data),
	}

	_, _, err = k.producer.SendMessage(msg)
	if err != nil {
		k.logger.Error("failed to produce message", err)
		return err
	}

	return nil
}

func (k *KafkaProducer) ProduceBatch(
	ctx context.Context,
	weathers []models.WeatherEntity,
) error {

	if len(weathers) == 0 {
		return nil
	}

	messages := make([]*sarama.ProducerMessage, 0, len(weathers))

	for _, weather := range weathers {
		data, err := json.Marshal(weather)
		if err != nil {
			return err
		}

		messages = append(messages, &sarama.ProducerMessage{
			Topic: k.topic,
			Value: sarama.ByteEncoder(data),
		})
	}

	if err := k.producer.SendMessages(messages); err != nil {
		k.logger.Error("failed to produce batch", err)
		return err
	}

	return nil
}

func (k *KafkaProducer) HealthCheck(ctx context.Context) error {
	if k.producer == nil {
		return errors.New("kafka producer is nil")
	}

	msg := &sarama.ProducerMessage{
		Topic: "__healthcheck",
		Value: sarama.ByteEncoder([]byte("ping")),
	}

	_, _, err := k.producer.SendMessage(msg)
	return err
}

func (k *KafkaProducer) Close() error {
	if k.producer == nil {
		return nil
	}
	return k.producer.Close()
}
