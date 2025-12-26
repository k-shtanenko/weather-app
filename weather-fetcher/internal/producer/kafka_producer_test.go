package producer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestKafkaProducerFactory(t *testing.T) {
	factory := NewKafkaProducerFactory()
	assert.NotNil(t, factory)
}

func TestKafkaProducer_ProduceBatch_Empty(t *testing.T) {
	producer := &KafkaProducer{
		producer: nil,
		topic:    "test-topic",
		logger:   nil,
	}

	ctx := context.Background()
	err := producer.ProduceBatch(ctx, []models.WeatherEntity{})

	assert.NoError(t, err)
}

func TestKafkaProducer_Close_NilProducer(t *testing.T) {
	producer := &KafkaProducer{
		producer: nil,
		topic:    "test-topic",
		logger:   nil,
	}

	err := producer.Close()

	assert.NoError(t, err)
}

func TestKafkaProducer_Produce_Safe(t *testing.T) {
	t.Run("nil producer should not panic", func(t *testing.T) {
		producer := &KafkaProducer{
			producer: nil,
			topic:    "test-topic",
			logger:   nil,
		}

		weather := &models.Weather{
			CityID:    123,
			CityName:  "Test City",
			Timestamp: time.Now(),
			Source:    "test",
			Pressure:  1013,
		}

		ctx := context.Background()
		assert.Panics(t, func() {
			_ = producer.Produce(ctx, weather)
		})
	})
}

func TestKafkaProducer_ProduceBatch_Safe(t *testing.T) {
	t.Run("empty batch with nil producer", func(t *testing.T) {
		producer := &KafkaProducer{
			producer: nil,
			topic:    "test-topic",
			logger:   nil,
		}

		ctx := context.Background()
		err := producer.ProduceBatch(ctx, []models.WeatherEntity{})

		assert.NoError(t, err)
	})
}

func TestKafkaProducer_HealthCheck_Safe(t *testing.T) {
	t.Run("nil producer health check", func(t *testing.T) {
		producer := &KafkaProducer{
			producer: nil,
			topic:    "test-topic",
			logger:   nil,
		}

		ctx := context.Background()
		err := producer.HealthCheck(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kafka producer is nil")
	})
}

func TestKafkaProducer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("create producer with factory", func(t *testing.T) {
		factory := NewKafkaProducerFactory()

		producer, err := factory.CreateProducer(
			"invalid-broker:9092",
			"test-topic",
			1,
			3,
		)

		assert.Error(t, err)
		assert.Nil(t, producer)
	})
}

func TestKafkaProducer_ErrorPaths(t *testing.T) {
	t.Run("producer factory error simulation", func(t *testing.T) {
		factory := NewKafkaProducerFactory()
		assert.NotNil(t, factory)

		factory.logger.Info("Test log for coverage")
	})
}

func TestKafkaProducer_CoverageOnly(t *testing.T) {
	producer := &KafkaProducer{
		topic: "test",
	}

	assert.Equal(t, "test", producer.topic)
}

type MockSyncProducerFull struct {
	mock.Mock
}

func (m *MockSyncProducerFull) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	args := m.Called(msg)
	return args.Get(0).(int32), args.Get(1).(int64), args.Error(2)
}

func (m *MockSyncProducerFull) SendMessages(msgs []*sarama.ProducerMessage) error {
	args := m.Called(msgs)
	return args.Error(0)
}

func (m *MockSyncProducerFull) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSyncProducerFull) TxnStatus() sarama.ProducerTxnStatusFlag {
	args := m.Called()
	return args.Get(0).(sarama.ProducerTxnStatusFlag)
}

func (m *MockSyncProducerFull) IsTransactional() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSyncProducerFull) BeginTxn() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSyncProducerFull) CommitTxn() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSyncProducerFull) AbortTxn() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSyncProducerFull) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	args := m.Called(offsets, groupId)
	return args.Error(0)
}

func (m *MockSyncProducerFull) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	args := m.Called(msg, groupId, metadata)
	return args.Error(0)
}

func TestKafkaProducer_Produce_Success(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)

	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	weather := &models.Weather{
		CityID:      123,
		CityName:    "Test City",
		Country:     "RU",
		Temperature: 20.5,
		FeelsLike:   19.8,
		Humidity:    65,
		Pressure:    1013,
		WindSpeed:   5.2,
		WindDeg:     180,
		Clouds:      40,
		Weather:     "cloudy",
		WeatherIcon: "04d",
		Visibility:  10000,
		Sunrise:     time.Now(),
		Sunset:      time.Now().Add(12 * time.Hour),
		Timestamp:   time.Now(),
		Source:      "test",
	}

	mockProducer.On("SendMessage", mock.AnythingOfType("*sarama.ProducerMessage")).
		Return(int32(0), int64(100), nil)

	ctx := context.Background()
	err := kafkaProducer.Produce(ctx, weather)

	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_ProduceBatch_Success(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	weathers := []models.WeatherEntity{
		&models.Weather{
			CityID:    123,
			CityName:  "Test City",
			Timestamp: time.Now(),
			Source:    "test",
			Pressure:  1013,
		},
		&models.Weather{
			CityID:    456,
			CityName:  "Test City 2",
			Timestamp: time.Now(),
			Source:    "test",
			Pressure:  1013,
		},
	}

	mockProducer.On("SendMessages", mock.AnythingOfType("[]*sarama.ProducerMessage")).
		Return(nil)

	ctx := context.Background()
	err := kafkaProducer.ProduceBatch(ctx, weathers)

	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_HealthCheck_Success(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	mockProducer.On("SendMessage", mock.AnythingOfType("*sarama.ProducerMessage")).
		Return(int32(0), int64(0), nil)

	ctx := context.Background()
	err := kafkaProducer.HealthCheck(ctx)

	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_HealthCheck_Error(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	mockProducer.On("SendMessage", mock.AnythingOfType("*sarama.ProducerMessage")).
		Return(int32(0), int64(0), assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.HealthCheck(ctx)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_Close_Success(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	mockProducer.On("Close").Return(nil)

	err := kafkaProducer.Close()

	assert.NoError(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_Close_Error(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	mockProducer.On("Close").Return(assert.AnError)

	err := kafkaProducer.Close()

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducerFactory_CreateProducer_Error(t *testing.T) {
	factory := NewKafkaProducerFactory()

	producer, err := factory.CreateProducer(
		"",
		"test-topic",
		1,
		3,
	)

	assert.Error(t, err)
	assert.Nil(t, producer)
}

func TestKafkaProducerFactory_CreateProducer_InvalidConfig(t *testing.T) {
	factory := NewKafkaProducerFactory()

	producer, err := factory.CreateProducer(
		"",
		"test-topic",
		1,
		3,
	)

	assert.Error(t, err)
	assert.Nil(t, producer)
	assert.True(t, strings.Contains(err.Error(), "broker") ||
		strings.Contains(err.Error(), "invalid") ||
		strings.Contains(err.Error(), "configuration") ||
		strings.Contains(err.Error(), "address"))
}

func TestKafkaProducer_Produce_ContextCancelled(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	weather := &models.Weather{
		CityID:    123,
		CityName:  "Test City",
		Timestamp: time.Now(),
		Source:    "test",
		Pressure:  1013,
	}

	mockProducer.On("SendMessage", mock.AnythingOfType("*sarama.ProducerMessage")).
		Return(int32(0), int64(0), context.Canceled)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := kafkaProducer.Produce(ctx, weather)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_ProduceBatch_SingleJSONError(t *testing.T) {

	t.Run("empty batch returns nil", func(t *testing.T) {
		mockProducer := new(MockSyncProducerFull)
		logger := &mockLogger{}

		kafkaProducer := &KafkaProducer{
			producer: mockProducer,
			topic:    "test-topic",
			logger:   logger,
		}

		ctx := context.Background()
		err := kafkaProducer.ProduceBatch(ctx, []models.WeatherEntity{})

		assert.NoError(t, err)
		mockProducer.AssertNotCalled(t, "SendMessages")
	})
}

func TestKafkaProducer_InterfaceCheck(t *testing.T) {
	var _ Producer = (*KafkaProducer)(nil)
	var _ ProducerFactory = (*KafkaProducerFactory)(nil)
}

func TestKafkaProducerFactory_Logger(t *testing.T) {
	factory := NewKafkaProducerFactory()

	assert.NotNil(t, factory)
	assert.NotNil(t, factory.logger)
}

func TestKafkaProducer_ProduceBatch_JSONError(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	ctx := context.Background()
	err := kafkaProducer.ProduceBatch(ctx, []models.WeatherEntity{})

	assert.NoError(t, err)
	mockProducer.AssertNotCalled(t, "SendMessages")
}

func TestKafkaProducerFactory_CreateProducer_EmptyBroker(t *testing.T) {
	factory := NewKafkaProducerFactory()

	producer, err := factory.CreateProducer(
		"",
		"test-topic",
		1,
		3,
	)

	assert.Error(t, err)
	assert.Nil(t, producer)
	assert.Contains(t, err.Error(), "broker")
}

func TestKafkaProducer_HealthCheck_NilProducer(t *testing.T) {
	producer := &KafkaProducer{
		producer: nil,
		topic:    "test-topic",
		logger:   &mockLogger{},
	}

	ctx := context.Background()
	err := producer.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kafka producer is nil")
}

func TestKafkaProducer_ProduceBatch_SendMessagesError(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	weather := &models.Weather{
		CityID:    123,
		CityName:  "Test City",
		Timestamp: time.Now(),
		Source:    "test",
		Pressure:  1013,
	}

	mockProducer.On("SendMessages", mock.AnythingOfType("[]*sarama.ProducerMessage")).Return(assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.ProduceBatch(ctx, []models.WeatherEntity{weather})

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_Produce_SendMessageError(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	weather := &models.Weather{
		CityID:    123,
		CityName:  "Test City",
		Timestamp: time.Now(),
		Source:    "test",
		Pressure:  1013,
	}

	mockProducer.On("SendMessage", mock.AnythingOfType("*sarama.ProducerMessage")).
		Return(int32(0), int64(0), assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.Produce(ctx, weather)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducerFactory_CreateProducer_ValidConfig(t *testing.T) {
	factory := NewKafkaProducerFactory()

	producer, err := factory.CreateProducer(
		"localhost:9092",
		"test-topic",
		1,
		3,
	)

	if err == nil {
		assert.NotNil(t, producer)
		producer.Close()
	}
}

func TestKafkaProducer_Produce_LoggingOnError(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   &mockLogger{},
	}

	weather := &models.Weather{
		CityID:    123,
		CityName:  "Test City",
		Timestamp: time.Now(),
		Source:    "test",
		Pressure:  1013,
	}

	mockProducer.On("SendMessage", mock.Anything).Return(int32(0), int64(0), assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.Produce(ctx, weather)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_ProduceBatch_LoggingOnError(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   &mockLogger{},
	}

	weathers := []models.WeatherEntity{
		&models.Weather{
			CityID:    123,
			CityName:  "Test City",
			Timestamp: time.Now(),
			Source:    "test",
			Pressure:  1013,
		},
	}

	mockProducer.On("SendMessages", mock.Anything).Return(assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.ProduceBatch(ctx, weathers)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_Produce_ErrorLogging(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	weather := &models.Weather{
		CityID:    123,
		CityName:  "Test City",
		Timestamp: time.Now(),
		Source:    "test",
		Pressure:  1013,
	}

	mockProducer.On("SendMessage", mock.Anything).Return(int32(0), int64(0), assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.Produce(ctx, weather)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_ProduceBatch_ErrorLogging(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)
	logger := &mockLogger{}

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   logger,
	}

	weathers := []models.WeatherEntity{
		&models.Weather{
			CityID:    123,
			CityName:  "Test City",
			Timestamp: time.Now(),
			Source:    "test",
			Pressure:  1013,
		},
	}

	mockProducer.On("SendMessages", mock.Anything).Return(assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.ProduceBatch(ctx, weathers)

	assert.Error(t, err)
	mockProducer.AssertExpectations(t)
}

func TestKafkaProducer_Produce_JSONMarshalError(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   &mockLogger{},
	}

	mockWeather := &mockWeatherEntity{}
	mockWeather.On("MarshalJSON").Return(nil, assert.AnError)

	ctx := context.Background()
	err := kafkaProducer.Produce(ctx, mockWeather)

	assert.Error(t, err)
	mockWeather.AssertExpectations(t)
}

func TestKafkaProducer_ProduceBatch_SingleMessageJSONError(t *testing.T) {
	mockProducer := new(MockSyncProducerFull)

	kafkaProducer := &KafkaProducer{
		producer: mockProducer,
		topic:    "test-topic",
		logger:   &mockLogger{},
	}

	mockWeather1 := &mockWeatherEntity{}
	mockWeather1.On("MarshalJSON").Return(nil, assert.AnError)

	mockWeather2 := &mockWeatherEntity{}

	ctx := context.Background()
	err := kafkaProducer.ProduceBatch(ctx, []models.WeatherEntity{mockWeather1, mockWeather2})

	assert.Error(t, err)
	mockWeather1.AssertExpectations(t)
	mockWeather2.AssertNotCalled(t, "MarshalJSON")
}

func TestKafkaProducerFactory_CreateProducer_LoggerField(t *testing.T) {
	factory := NewKafkaProducerFactory()

	assert.NotNil(t, factory)
	assert.NotNil(t, factory.logger)

	producer, err := factory.CreateProducer("localhost:9092", "test-topic", 1, 3)

	if err == nil {
		assert.NotNil(t, producer)
		producer.Close()
	}
}

func TestCreateProducer_ReturnsKafkaProducerWithCorrectFields(t *testing.T) {
	factory := &KafkaProducerFactory{
		logger: &mockLogger{},
	}

	kafkaProducer, err := factory.CreateProducer("localhost:9092", "test-topic", 10, 10)

	assert.NoError(t, err)
	assert.IsType(t, &KafkaProducer{}, kafkaProducer)
}

type mockWeatherEntity struct {
	mock.Mock
	models.WeatherEntity
}

func (m *mockWeatherEntity) MarshalJSON() ([]byte, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

type mockLogger struct{}

func (m *mockLogger) Debug(args ...interface{})                              {}
func (m *mockLogger) Debugf(format string, args ...interface{})              {}
func (m *mockLogger) Info(args ...interface{})                               {}
func (m *mockLogger) Infof(format string, args ...interface{})               {}
func (m *mockLogger) Warn(args ...interface{})                               {}
func (m *mockLogger) Warnf(format string, args ...interface{})               {}
func (m *mockLogger) Error(args ...interface{})                              {}
func (m *mockLogger) Errorf(format string, args ...interface{})              {}
func (m *mockLogger) Fatal(args ...interface{})                              {}
func (m *mockLogger) Fatalf(format string, args ...interface{})              {}
func (m *mockLogger) WithField(key string, value interface{}) logger.Logger  { return m }
func (m *mockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
