package eventually

import (
	"context"
	"fmt"
	"github.com/ahmedkamals/eventually/internal/errors"
	"reflect"
	"time"
)

type (
	producer interface {
		// Publish topic(s).
		Publish(...Descriptor)
		// PublishAfter specified duration a topic(s).
		PublishAfter(context.Context, time.Duration, ...Descriptor)
	}

	scheduler interface {
		// SchedulePublish for a topic(s) when every specified duration elapses.
		// The ticker should Run infinitely during the application life time, unless got context timeout.
		SchedulePublish(context.Context, *time.Ticker, ...Descriptor)
	}

	runner interface {
		// Run event bus operations.
		Run(context.Context)
	}

	// Bus implements publish/subscribe pattern: https://en.wikipedia.org/wiki/Publish%E2%80%93subscribe_pattern
	Bus interface {
		producer
		scheduler
		runner
		// Topics returns the current registered topicsMap.
		Topics() []Descriptor
		// Length returns the number of subscribed topicsMap.
		Length() int
		// Subscribe a certain consumer to a single or a group of topicsMap.
		Subscribe(Consumer, ...Descriptor) error
		// SubscribeToAll topicsMap.
		SubscribeToAll(Consumer) error
		// Unsubscribe removes a consumer for a group of topicsMap.
		Unsubscribe(Consumer, ...Descriptor) error
		// Unregister removes all consumers for the specified topicsMap.
		Unregister(...Descriptor) error
	}

	// Logger interface for logging operations.
	Logger interface {
		Log(string)
	}

	// ErrorQueue interface for error reporting.
	ErrorQueue interface {
		Report(error)
	}

	// bus struct
	bus struct {
		eventStore     EventStore
		dispatcher     Dispatcher
		registerChan   chan subscription
		unRegisterChan chan subscription
		logger         Logger
		errorQueue     ErrorQueue
	}
)

func (b *bus) Topics() []Descriptor {
	return b.eventStore.Topics()
}

func (b *bus) Length() int {
	return b.eventStore.Length()
}

func (b *bus) Publish(topics ...Descriptor) {
	b.dispatcher.dispatch(topics...)
}

func (b *bus) PublishAfter(ctx context.Context, duration time.Duration, topics ...Descriptor) {
	go b.dispatcher.publishAfter(ctx, duration, topics...)
}

func (b *bus) SchedulePublish(ctx context.Context, ticker *time.Ticker, topics ...Descriptor) {
	go b.dispatcher.schedulePublish(ctx, ticker, topics...)
}

func (b *bus) subscribe(ctx context.Context, consumer Consumer, topics ...Descriptor) {
	const op errors.Operation = "Bus.subscribe"

	defer func() {
		if err := recover(); err != nil {
			b.errorQueue.Report(errors.E(op, errors.Panic, err))
		}
	}()

	for _, topic := range topics {
		select {
		case <-ctx.Done():
			break
		default:
		}

		if topic == nil {
			b.errorQueue.Report(errors.E(op, errors.Invalid))
			continue
		}

		if b.eventStore.IsSubscribed(topic, consumer) {
			b.errorQueue.Report(errors.E(op, errors.Exist, errors.Errorf("%s", topic)))
			continue
		}

		b.eventStore.Store(topic, consumer)

		consumer.OnSubscribe(topic, func(topic Descriptor, consumer Consumer) {
			b.logger.Log(fmt.Sprintf("Topic %s subscription is announced by %s.OnSubscribe", topic, consumer))
		})
	}
}

func (b *bus) Subscribe(consumer Consumer, topics ...Descriptor) error {
	const op errors.Operation = "Bus.Subscribe"

	if consumer == nil || reflect.ValueOf(consumer).IsNil() {
		return errors.E(op, errors.Invalid)
	}

	if len(topics) == 0 {
		return errors.E(op, errors.MinLength)
	}

	go func() {
		b.registerChan <- newNormalSubscription(consumer, topics...)
	}()

	return nil
}

func (b *bus) SubscribeToAll(consumer Consumer) error {
	const op errors.Operation = "Bus.SubscribeToAll"

	if consumer == nil || reflect.ValueOf(consumer).IsNil() {
		return errors.E(op, errors.Invalid)
	}

	go func() {
		b.registerChan <- newUniversalSubscription(consumer)
	}()

	return nil
}

func (b *bus) unsubscribe(consumer Consumer, topics ...Descriptor) {
	const op errors.Operation = "Bus.unsubscribe"

	defer func() {
		if err := recover(); err != nil {
			b.errorQueue.Report(errors.E(op, errors.Panic, err))
		}
	}()

	for _, topic := range topics {
		if !b.eventStore.Remove(topic, consumer) {
			b.errorQueue.Report(errors.E(op, errors.NotFound, errors.Errorf(" for consumer %#v to %s", consumer, topic)))
			continue
		}

		consumer.Signout()
		consumer.OnUnsubscribe(topic, func(topic Descriptor, consumer Consumer) {
			b.logger.Log(fmt.Sprintf("Topic %s subscription cancellation is announced by %s.OnUnubscribe", topic, consumer))
		})
	}
}

func (b *bus) Unsubscribe(consumer Consumer, topics ...Descriptor) error {
	const op errors.Operation = "Bus.Unsubscribe"

	if len(topics) == 0 {
		return errors.E(op, errors.MinLength)
	}

	go func() {
		b.unRegisterChan <- newNormalSubscription(consumer, topics...)
	}()

	return nil
}

func (b *bus) unregister(topics ...Descriptor) {
	const op errors.Operation = "Bus.unregister"

	defer func() {
		if err := recover(); err != nil {
			b.errorQueue.Report(errors.E(op, errors.Panic, err))
		}
	}()

	for _, topic := range topics {
		b.eventStore.DeleteTopic(topic)
	}
}

func (b *bus) Unregister(topics ...Descriptor) error {
	const op errors.Operation = "bus.Unregister"

	if len(topics) == 0 {
		return errors.E(op, errors.MinLength)
	}

	go func() {
		b.logger.Log(fmt.Sprintf("Unregistering topicsMap %s", topics))
		b.unRegisterChan <- newNormalSubscription(nil, topics...)
	}()

	return nil
}

// monitorRegistration of consumers to topicsMap.
func (b *bus) monitorRegistration(controlCtx context.Context) {
	const op errors.Operation = "Bus.monitorRegistration"

	defer func() {
		if err := recover(); err != nil {
			b.errorQueue.Report(errors.E(op, errors.Panic, err))
		}
	}()

	for {
		select {
		case subscription := <-b.registerChan:
			if subscription.isUniversal() {
				go b.subscribe(controlCtx, subscription.getConsumer(), newUniversalDescriptor("universalDescriptor", nil, 1))
				return
			}
			go b.subscribe(controlCtx, subscription.getConsumer(), subscription.getTopics()...)
		case <-controlCtx.Done():
			return
		}
	}
}

// monitorUnRegistration forwards subscriptions cancellation for specific/all topicsMap.
func (b *bus) monitorUnRegistration(controlCtx context.Context) {
	const op errors.Operation = "Bus.monitorUnRegistration"

	defer func() {
		if err := recover(); err != nil {
			b.errorQueue.Report(errors.E(op, errors.Panic, err))
		}
	}()

	for {
		select {
		case subscription := <-b.unRegisterChan:

			if subscription.getConsumer() != nil {
				go b.unsubscribe(subscription.getConsumer(), subscription.getTopics()...)
				continue
			}

			go b.unregister(subscription.getTopics()...)
		case <-controlCtx.Done():
			return
		}
	}
}

// Run the bus operations.
func (b *bus) Run(controlCtx context.Context) {
	go b.monitorRegistration(controlCtx)
	go b.dispatcher.monitorDispatch(controlCtx)
	go b.monitorUnRegistration(controlCtx)
}

// NewBus creates new Bus
func NewBus(eventStore EventStore, logger Logger, errorQueue ErrorQueue, bufferSize int) Bus {
	eventBus := &bus{
		eventStore:     eventStore,
		dispatcher:     NewDispatcher(eventStore, logger, errorQueue, bufferSize),
		registerChan:   make(chan subscription, bufferSize),
		unRegisterChan: make(chan subscription, bufferSize),
		logger:         logger,
		errorQueue:     errorQueue,
	}

	return eventBus
}
