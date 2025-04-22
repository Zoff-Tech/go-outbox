package broker

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"
	"github.com/zoff-tech/go-outbox/pkg/config"
)

type pooledChannel struct {
	channel     *amqp.Channel
	notifyClose chan *amqp.Error
}

func newConnection(settings *config.BrokerSettings) (*amqp.Connection, error) {
	conn, err := amqp.Dial(settings.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	// Set up a channel to handle connection close notifications
	notifyClose := make(chan *amqp.Error)
	conn.NotifyClose(notifyClose)
	go func() {
		for err := range notifyClose {
			log.Printf("RabbitMQ connection closed: %v", err)
		}
	}()

	return conn, nil
}

func (r *rabbitMqBroker) connectAndInitialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Close existing connection if it exists
	if r.connection != nil && !r.connection.IsClosed() {
		r.connection.Close()
	}

	// Establish a new connection
	connection, err := newConnection(r.settings)

	if err != nil {
		return err
	}
	r.connection = connection

	// Clear the existing channel pool
	close(r.channelPool)
	r.channelPool = make(chan *pooledChannel, r.settings.PoolSize)

	// Declare the exchange
	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// Reinitialize the channel pool
	for i := 0; i < r.settings.PoolSize; i++ {
		channel, err := connection.Channel()
		if err != nil {
			return err
		}
		r.channelPool <- &pooledChannel{
			channel:     channel,
			notifyClose: channel.NotifyClose(make(chan *amqp.Error)),
		}
	}

	log.Println("RabbitMQ connection, exchange, and channel pool initialized")
	return nil
}

func (r *rabbitMqBroker) recoverConnection() {
	for {
		select {
		case <-r.reconnectTicker.C:
			if r.connection == nil || r.connection.IsClosed() {
				log.Println("Attempting to reconnect to RabbitMQ...")
				if err := r.connectAndInitialize(); err != nil {
					log.Printf("Failed to reconnect to RabbitMQ: %v", err)
				} else {
					log.Println("Reconnected to RabbitMQ successfully")
				}
			}
		case <-r.stopReconnect:
			log.Println("Stopping RabbitMQ connection recovery")
			return
		}
	}
}

func (r *rabbitMqBroker) getChannel() (*pooledChannel, error) {
	for {
		select {
		case pooledChan := <-r.channelPool:
			select {
			case err := <-pooledChan.notifyClose:
				// Channel is closed, discard it
				log.Printf("Discarding closed channel: %v", err)
				continue
			default:
				// Channel is valid
				log.Println("Retrieved channel from pool")
				return pooledChan, nil
			}
		default:
			// Create a new channel if none are available
			log.Println("Creating new channel")
			channel, err := r.connection.Channel()
			if err != nil {
				return nil, err
			}
			return &pooledChannel{
				channel:     channel,
				notifyClose: channel.NotifyClose(make(chan *amqp.Error)),
			}, nil
		}
	}
}

func (r *rabbitMqBroker) releaseChannel(pooledChan *pooledChannel) {
	select {
	case err := <-pooledChan.notifyClose:
		// Channel is closed, discard it
		log.Printf("Discarding closed channel: %v", err)
		return
	default:
		// Channel is valid, return it to the pool
		select {
		case r.channelPool <- pooledChan:
			log.Println("Returned channel to pool")
		default:
			// Pool is full, close the channel
			log.Println("Closing channel as pool is full")
			pooledChan.channel.Close()
		}
	}
}
