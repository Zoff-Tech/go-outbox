package broker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zoff-tech/go-outbox/config"
	"github.com/zoff-tech/go-outbox/store"
)

// --- Mocks ---

type mockAmqpConnection struct {
	mock.Mock
	closed bool
}

func (m *mockAmqpConnection) Channel() (*amqp.Channel, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*amqp.Channel), args.Error(1)
}
func (m *mockAmqpConnection) Close() error {
	m.closed = true
	return m.Called().Error(0)
}
func (m *mockAmqpConnection) IsClosed() bool {
	return m.closed
}

type mockChannel struct {
	mock.Mock
	closed bool
}

func (m *mockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return m.Called(name, kind, durable, autoDelete, internal, noWait, args).Error(0)
}
func (m *mockChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return m.Called(exchange, key, mandatory, immediate, msg).Error(0)
}
func (m *mockChannel) Close() error {
	m.closed = true
	return m.Called().Error(0)
}
func (m *mockChannel) NotifyClose(c chan *amqp.Error) chan *amqp.Error {
	return c
}

// --- Tests ---

func newTestBroker(poolSize int, conn *mockAmqpConnection, ch *mockChannel) *rabbitMqBroker {
	b := &rabbitMqBroker{
		connection:      conn,
		channelPool:     make(chan *pooledChannel, poolSize),
		settings:        &config.BrokerSettings{PoolSize: poolSize},
		reconnectTicker: time.NewTicker(time.Hour), // never fires
		stopReconnect:   make(chan struct{}),
	}
	for i := 0; i < poolSize; i++ {
		b.channelPool <- &pooledChannel{
			channel:     ch,
			notifyClose: make(chan *amqp.Error, 1),
		}
	}
	return b
}

func TestPublish_Success(t *testing.T) {
	conn := new(mockAmqpConnection)
	ch := new(mockChannel)
	broker := newTestBroker(1, conn, ch)

	ch.On("ExchangeDeclare", "ex", "direct", true, false, false, false, mock.Anything).Return(nil)
	ch.On("Publish", "ex", "rk", false, false, mock.Anything).Return(nil)

	event := &store.OutboxEvent{
		Entity:     "ex",
		EntityType: "direct",
		RoutingKey: "rk",
		Payload:    []byte("payload"),
		Headers:    map[string]string{"foo": "bar"},
	}
	err := broker.Publish(context.Background(), event)
	assert.NoError(t, err)
	ch.AssertExpectations(t)
}

func TestPublish_ExchangeDeclareError(t *testing.T) {
	conn := new(mockAmqpConnection)
	ch := new(mockChannel)
	broker := newTestBroker(1, conn, ch)

	ch.On("ExchangeDeclare", "ex", "direct", true, false, false, false, mock.Anything).Return(errors.New("exch"))
	event := &store.OutboxEvent{
		Entity:     "ex",
		EntityType: "direct",
		RoutingKey: "rk",
		Payload:    []byte("payload"),
		Headers:    map[string]string{},
	}
	err := broker.Publish(context.Background(), event)
	assert.ErrorContains(t, err, "exch")
	ch.AssertExpectations(t)
}

func TestPublish_PublishError(t *testing.T) {
	conn := new(mockAmqpConnection)
	ch := new(mockChannel)
	broker := newTestBroker(1, conn, ch)

	ch.On("ExchangeDeclare", "ex", "direct", true, false, false, false, mock.Anything).Return(nil)
	ch.On("Publish", "ex", "rk", false, false, mock.Anything).Return(errors.New("pub"))
	event := &store.OutboxEvent{
		Entity:     "ex",
		EntityType: "direct",
		RoutingKey: "rk",
		Payload:    []byte("payload"),
		Headers:    map[string]string{},
	}
	err := broker.Publish(context.Background(), event)
	assert.ErrorContains(t, err, "pub")
	ch.AssertExpectations(t)
}

func TestPublish_GetChannelError(t *testing.T) {
	conn := new(mockAmqpConnection)
	broker := &rabbitMqBroker{
		connection:      conn,
		channelPool:     make(chan *pooledChannel, 0),
		settings:        &config.BrokerSettings{PoolSize: 1},
		reconnectTicker: time.NewTicker(time.Hour),
		stopReconnect:   make(chan struct{}),
	}
	conn.On("Channel").Return(nil, errors.New("chanfail"))
	event := &store.OutboxEvent{
		Entity:     "ex",
		EntityType: "direct",
		RoutingKey: "rk",
		Payload:    []byte("payload"),
		Headers:    map[string]string{},
	}
	err := broker.Publish(context.Background(), event)
	assert.ErrorContains(t, err, "chanfail")
}

func TestReleaseChannel_Closed(t *testing.T) {
	conn := new(mockAmqpConnection)
	ch := new(mockChannel)
	broker := newTestBroker(1, conn, ch)
	pooled := &pooledChannel{
		channel:     ch,
		notifyClose: make(chan *amqp.Error, 1),
	}
	pooled.notifyClose <- amqp.ErrClosed
	broker.releaseChannel(pooled)
	// Should discard, not panic
}

func TestReleaseChannel_PoolFull(t *testing.T) {
	conn := new(mockAmqpConnection)
	ch := new(mockChannel)
	broker := newTestBroker(1, conn, ch)
	pooled := &pooledChannel{
		channel:     ch,
		notifyClose: make(chan *amqp.Error, 1),
	}
	ch.On("Close").Return(nil)
	// Do NOT pre-fill the pool, just call releaseChannel (the pool is already full from newTestBroker)
	broker.releaseChannel(pooled)
	ch.AssertExpectations(t)
}

func TestClose(t *testing.T) {
	conn := new(mockAmqpConnection)
	ch := new(mockChannel)
	broker := newTestBroker(1, conn, ch)
	ch.On("Close").Return(nil)
	conn.On("Close").Return(nil)
	err := broker.Close()
	assert.NoError(t, err)
	ch.AssertExpectations(t)
	conn.AssertExpectations(t)
}

func TestRecoverConnection_Stop(t *testing.T) {
	conn := new(mockAmqpConnection)
	broker := &rabbitMqBroker{
		connection:      conn,
		channelPool:     make(chan *pooledChannel, 1),
		settings:        &config.BrokerSettings{PoolSize: 1},
		reconnectTicker: time.NewTicker(10 * time.Millisecond),
		stopReconnect:   make(chan struct{}),
	}
	done := make(chan struct{})
	go func() {
		broker.recoverConnection()
		close(done)
	}()
	close(broker.stopReconnect)
	<-done
}
func TestConnectAndInitialize_ChannelError(t *testing.T) {
	conn := new(mockAmqpConnection)
	broker := &rabbitMqBroker{
		connection:      conn,
		channelPool:     make(chan *pooledChannel, 1),
		settings:        &config.BrokerSettings{PoolSize: 1, URL: "amqp://test"},
		reconnectTicker: time.NewTicker(time.Hour),
		stopReconnect:   make(chan struct{}),
	}
	// Patch newConnection to return our mock
	origNewConnection := newConnection
	defer func() { newConnection = origNewConnection }()
	newConnection = func(settings *config.BrokerSettings) (*amqp.Connection, error) {
		return nil, errors.New("failconn")
	}
	conn.On("Close").Return(nil) // <-- Add this line
	err := broker.connectAndInitialize()
	assert.ErrorContains(t, err, "failconn")
}
