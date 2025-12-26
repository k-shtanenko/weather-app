package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/k-shtanenko/weather-app/weather-api/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-api/internal/models"
)

type Consumer interface {
	Consume(ctx context.Context, handler func(ctx context.Context, message models.WeatherDataEntity) error) error
	Close() error
	HealthCheck(ctx context.Context) error
}

type KafkaConsumer struct {
	consumer sarama.ConsumerGroup
	topic    string
	groupID  string
	logger   logger.Logger
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

type consumerHandler struct {
	handler func(ctx context.Context, message models.WeatherDataEntity) error
	logger  logger.Logger
}

func NewKafkaConsumer(broker string, topic, groupID string, batchSize int) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRange()
	config.Consumer.MaxProcessingTime = 30 * time.Second
	config.Net.DialTimeout = 30 * time.Second
	config.Net.ReadTimeout = 30 * time.Second
	config.Net.WriteTimeout = 30 * time.Second

	consumer, err := sarama.NewConsumerGroup([]string{broker}, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &KafkaConsumer{
		consumer: consumer,
		topic:    topic,
		groupID:  groupID,
		logger:   logger.New("info", "development").WithField("component", "kafka_consumer"),
	}, nil
}

func (k *KafkaConsumer) Consume(ctx context.Context, handler func(ctx context.Context, message models.WeatherDataEntity) error) error {
	k.logger.Infof("Starting Kafka consumer for topic: %s, group: %s", k.topic, k.groupID)

	ctx, cancel := context.WithCancel(ctx)
	k.cancel = cancel

	consumerHandler := &consumerHandler{
		handler: handler,
		logger:  k.logger.WithField("handler", "kafka"),
	}

	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		for {
			select {
			case <-ctx.Done():
				k.logger.Info("Kafka consumer context cancelled, stopping...")
				return
			default:
				if err := k.consumer.Consume(ctx, []string{k.topic}, consumerHandler); err != nil {
					k.logger.Errorf("Error consuming from Kafka: %v", err)
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()

	go func() {
		for err := range k.consumer.Errors() {
			k.logger.Errorf("Kafka consumer error: %v", err)
		}
	}()

	k.logger.Info("Kafka consumer started successfully")
	return nil
}

func (k *KafkaConsumer) Close() error {
	k.logger.Info("Closing Kafka consumer...")

	if k.cancel != nil {
		k.cancel()
	}

	k.wg.Wait()

	if err := k.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka consumer: %w", err)
	}

	k.logger.Info("Kafka consumer closed successfully")
	return nil
}

func (k *KafkaConsumer) HealthCheck(ctx context.Context) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Net.DialTimeout = 5 * time.Second

	client, err := sarama.NewClient([]string{"kafka:9093"}, config)
	if err != nil {
		return fmt.Errorf("failed to create Kafka client: %w", err)
	}
	defer client.Close()

	topics, err := client.Topics()
	if err != nil {
		return fmt.Errorf("failed to get topics: %w", err)
	}

	found := false
	for _, topic := range topics {
		if topic == k.topic {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("topic %s not found", k.topic)
	}

	return nil
}

func (h *consumerHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer handler setup completed")
	return nil
}

func (h *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer handler cleanup")
	return nil
}

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.logger.Infof("Starting to consume claims for partition %d", claim.Partition())

	for message := range claim.Messages() {
		select {
		case <-session.Context().Done():
			h.logger.Info("Consumer session context done, stopping consumption")
			return nil
		default:
			weatherData, err := h.deserializeMessage(message.Value)
			if err != nil {
				h.logger.Errorf("Failed to deserialize message: %v", err)
				session.MarkMessage(message, "")
				continue
			}

			if err := h.handler(session.Context(), weatherData); err != nil {
				h.logger.Errorf("Failed to handle message: %v", err)
				continue
			}

			session.MarkMessage(message, "")
			h.logger.Debugf("Processed message for city: %s (offset: %d)", weatherData.GetCityName(), message.Offset)
		}
	}

	return nil
}

func (h *consumerHandler) deserializeMessage(data []byte) (models.WeatherDataEntity, error) {
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Kafka message: %w", err)
	}

	recordedAt, _ := time.Parse(time.RFC3339, rawData["timestamp"].(string))
	sunrise, _ := time.Parse(time.RFC3339, rawData["sunrise"].(string))
	sunset, _ := time.Parse(time.RFC3339, rawData["sunset"].(string))

	weatherData := &models.WeatherData{
		CityID:             int(rawData["city_id"].(float64)),
		CityName:           rawData["city_name"].(string),
		Country:            rawData["country"].(string),
		Temperature:        rawData["temperature"].(float64),
		FeelsLike:          rawData["feels_like"].(float64),
		Humidity:           int(rawData["humidity"].(float64)),
		Pressure:           int(rawData["pressure"].(float64)),
		WindSpeed:          rawData["wind_speed"].(float64),
		WindDeg:            int(rawData["wind_deg"].(float64)),
		Clouds:             int(rawData["clouds"].(float64)),
		WeatherDescription: rawData["weather"].(string),
		WeatherIcon:        rawData["weather_icon"].(string),
		Visibility:         int(rawData["visibility"].(float64)),
		Sunrise:            sunrise,
		Sunset:             sunset,
		RecordedAt:         recordedAt,
		Source:             rawData["source"].(string),
		CreatedAt:          time.Now(),
	}

	weatherData.ID = fmt.Sprintf("%d-%s", weatherData.CityID, recordedAt.Format("20060102150405"))

	if err := weatherData.Validate(); err != nil {
		return nil, fmt.Errorf("invalid weather data: %w", err)
	}

	return weatherData, nil
}
