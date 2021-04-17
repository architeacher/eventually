package eventually

import (
	"context"
	"fmt"
	"github.com/ahmedkamals/eventually/internal/errors"
	"time"
)

type (
	// Dispatcher is responsible for dispatching topics.
	Dispatcher interface {
		// monitorDispatch of topicsMap publishing requests.
		monitorDispatch(context.Context)
		// dispatch topic(s).
		dispatch(...Descriptor)
		// publishAfter drops a message at consumers mail boxes after specified duration.
		publishAfter(context.Context, time.Duration, ...Descriptor)
		// schedulePublish drops a message at consumers' mail boxes when every specified duration elapses.
		// The ticker should Run infinitely during the application life time, unless got context timeout.
		schedulePublish(context.Context, *time.Ticker, ...Descriptor)
	}

	// bus struct
	dispatcher struct {
		eventStore   EventStore
		dispatchChan chan Descriptor
		logger       Logger
		errorQueue   ErrorQueue
	}
)

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(eventStore EventStore, logger Logger, errorQueue ErrorQueue, bufferSize int) Dispatcher {
	return &dispatcher{
		eventStore:   eventStore,
		dispatchChan: make(chan Descriptor, bufferSize),
		logger:       logger,
		errorQueue:   errorQueue,
	}
}

func (d *dispatcher) getTopicConsumers(aggregateID UUID, topic Descriptor) []Consumer {
	consumers := make([]Consumer, 0)

	topicConsumers, ok := d.eventStore.Load(aggregateID)
	if !ok {
		return consumers
	}

	for topicConsumer := range topicConsumers.Iterator() {
		match := topicConsumer.MatchCriteria()
		if match != nil && !match(topic) {
			continue
		}
		consumers = append(consumers, topicConsumer)
	}

	return consumers
}

func (d *dispatcher) getMatchedConsumers(topic Descriptor) []Consumer {
	const op errors.Operation = "Dispatcher.getMatchedConsumers"

	defer func() {
		if err := recover(); err != nil {
			d.errorQueue.Report(errors.E(op, errors.Panic, err))
		}
	}()

	universalConsumers := d.getTopicConsumers(universalTopicUUID, nil)
	topicConsumers := d.getTopicConsumers(topic.AggregateID(), topic)
	if len(universalConsumers) == 0 {
		return topicConsumers
	}

	uniqueConsumers := make(map[Consumer]struct{})

	for _, consumer := range universalConsumers {
		uniqueConsumers[consumer] = struct{}{}
	}

	for _, consumer := range topicConsumers {
		uniqueConsumers[consumer] = struct{}{}
	}

	consumers := make([]Consumer, len(uniqueConsumers))

	index := 0
	for consumer := range uniqueConsumers {
		consumers[index] = consumer
		index++
	}

	return consumers
}

// publish a message and drops it at consumers mail boxes.
func (d *dispatcher) publish(topic Descriptor) {
	const op errors.Operation = "Dispatcher.publish"

	d.logger.Log(fmt.Sprintf("Publishing topic %s", topic))

	consumers := d.getMatchedConsumers(topic)

	if len(consumers) == 0 {
		return
	}

	for _, consumer := range consumers {

		go func(consumer Consumer) {
			defer func() {
				if err := recover(); err != nil {
					d.errorQueue.Report(errors.E(op, errors.Panic, err))
				}
			}()
			consumer.Drop(topic)
		}(consumer)
	}
}

func (d *dispatcher) publishOnTimeBasis(ctx context.Context, topic Descriptor, timeChan <-chan time.Time) {
	for {
		select {
		case <-timeChan:
			d.dispatch(topic)
		case <-ctx.Done():
			d.errorQueue.Report(ctx.Err())
			return
		}
	}
}

func (d *dispatcher) publishAfter(ctx context.Context, duration time.Duration, topics ...Descriptor) {
	for _, topic := range topics {
		go d.publishOnTimeBasis(ctx, topic, time.After(duration))
	}
}

func (d *dispatcher) schedulePublish(ctx context.Context, ticker *time.Ticker, topics ...Descriptor) {
	for _, topic := range topics {
		go d.publishOnTimeBasis(ctx, topic, ticker.C)
	}
}

// monitorDispatch of topicsMap to their relevant consumers.
func (d *dispatcher) monitorDispatch(controlCtx context.Context) {
	const op errors.Operation = "Dispatcher.monitorDispatch"

	defer func() {
		if err := recover(); err != nil {
			d.errorQueue.Report(errors.E(op, errors.Panic, err))
		}
	}()

	for {
		select {
		case topic := <-d.dispatchChan:
			d.publish(topic)
		case <-controlCtx.Done():
			return
		}
	}
}

func (d *dispatcher) dispatch(topics ...Descriptor) {
	for _, topic := range topics {
		go func(topic Descriptor) {
			d.dispatchChan <- topic
		}(topic)
	}
}
