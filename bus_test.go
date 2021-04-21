package eventually

import (
	"context"
	"fmt"
	"github.com/ahmedkamals/eventually/internal/errors"
	"github.com/stretchr/testify/assert"
	"strings"
	"sync"
	"testing"
	"time"
)

type (
	fakeEventLogger struct {
		logChan chan string
	}

	fakeErrorQueue struct {
		errChan chan error
	}

	fakeConsumer struct {
		sync.Mutex
		id                        UUID
		isOnSubscriptionTriggered bool
		descriptorChan            chan Descriptor
		mailBox                   Inbox
		shouldMatch               bool
	}
)

const (
	testBufferSize = 100
	delayTime      = 18 * time.Millisecond
)

func newFakeEventLogger(logChan chan string) Logger {
	return &fakeEventLogger{
		logChan: logChan,
	}
}

func (fel *fakeEventLogger) Log(message string) {
	select {
	case fel.logChan <- message:
	// Drop any log message that exceeds the log queue size.
	default:
	}
}

func (fel *fakeEventLogger) get() string {
	return <-fel.logChan
}

func newFakeErrorQueue(errChan chan error) ErrorQueue {
	return &fakeErrorQueue{
		errChan: errChan,
	}
}

func (feq *fakeErrorQueue) Report(err error) {
	select {
	case feq.errChan <- err:
	// Drop any error message that exceeds the error queue size.
	default:
	}
}

func (feq *fakeErrorQueue) getError() error {
	return <-feq.errChan
}

func newIdentifiableFakeConsumer(aggregateID UUID) *fakeConsumer {
	return &fakeConsumer{
		id:             aggregateID,
		mailBox:        newFakeInbox(testBufferSize),
		descriptorChan: make(chan Descriptor, testBufferSize),
		shouldMatch:    true,
	}
}

func newFakeConsumer() *fakeConsumer {
	return newIdentifiableFakeConsumer(NewUUID())
}

func (fc *fakeConsumer) AggregateID() UUID {
	return fc.id
}

func (fc *fakeConsumer) MatchCriteria() Match {
	return func(topic Descriptor) bool {
		return fc.shouldMatch
	}
}

func (fc *fakeConsumer) Drop(descriptor Descriptor) {
	fc.mailBox.Receive(descriptor)
}

func (fc *fakeConsumer) OnSubscribe(topic Descriptor, notification Notification) {
	fc.Lock()
	fc.isOnSubscriptionTriggered = true
	notification(topic, fc)
	fc.Unlock()
}

func (fc *fakeConsumer) OnUnsubscribe(topic Descriptor, notification Notification) {
	notification(topic, fc)
}

func (fc *fakeConsumer) ReadMessage() Descriptor {
	return fc.mailBox.Read()
}

func (fc *fakeConsumer) Signout() {
	fc.mailBox.Signout()
}

func (fc *fakeConsumer) GoString() string {
	return fmt.Sprintf("%s[%s]", fc.String(), fc.id)
}

func (fc *fakeConsumer) String() string {
	return "fakeConsumer"
}

func (fc *fakeConsumer) getOnSubscribeTriggerValue() bool {
	fc.Lock()
	defer fc.Unlock()

	return fc.isOnSubscriptionTriggered
}

func newEventBus(controlCtx context.Context, bufferSize int) *bus {
	logger := newFakeEventLogger(make(chan string, bufferSize))
	queue := newFakeErrorQueue(make(chan error, bufferSize))

	eventBus := NewBus(NewEventStore(), logger, queue, bufferSize).(*bus)
	eventBus.Run(controlCtx)

	return eventBus
}

func TestShouldGetTopics(t *testing.T) {
	t.Parallel()

	topics := []Descriptor{
		newTestDescriptor(),
		newTestDescriptor(),
		newTestDescriptor(),
	}

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected []Descriptor
	}{
		{
			"Should return empty topics.",
			map[string]interface{}{
				"topicsMap": map[Consumer][]Descriptor{},
			},
			[]Descriptor{},
		},
		{
			"Should return non duplicate existing topics.",
			map[string]interface{}{
				"topicsMap": map[Consumer][]Descriptor{
					newFakeConsumer(): topics,
				},
			},
			topics,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			topicsMap := testCase.input["topicsMap"].(map[Consumer][]Descriptor)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			eventBus := newEventBus(ctx, testBufferSize)

			for consumer, topics := range topicsMap {
				err := eventBus.Subscribe(consumer, topics...)
				if err != nil {
					t.Error(err)
				}
			}
			// Delay till the topics are subscribed.
			<-time.After(delayTime)
			assert.ElementsMatch(t, testCase.expected, eventBus.Topics(), 0, testCase.id)

			err := eventBus.Unregister(topics...)
			if err != nil {
				t.Error(err)
			}
			// Delay till the topics are unregistered.
			<-time.After(delayTime)

			assert.Equal(t, 0, len(eventBus.Topics()), testCase.id)
		})
	}
}

func TestPublish(t *testing.T) {
	t.Parallel()

	topic := newTestDescriptor()

	testCases := []struct {
		id                 string
		input              map[string]interface{}
		expectedTopic      Descriptor
		expectedLogMessage string
	}{
		{
			"Publish should be done correctly.",
			map[string]interface{}{
				"topic": topic,
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			topic,
			"Publishing topic testTopic@v1",
		},
		{
			"Publish should be done correctly when there are no consumers.",
			map[string]interface{}{
				"consumers": []Consumer{},
				"topic":     topic,
			},
			topic,
			"Publishing topic testTopic@v1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			topic := testCase.input["topic"].(Descriptor)
			consumers := testCase.input["consumers"].([]Consumer)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			eventBus := newEventBus(ctx, testBufferSize)

			for _, consumer := range consumers {
				err := eventBus.Subscribe(consumer, topic)
				if err != nil {
					t.Error(err)
				}
			}
			// Delay till the consumers are subscribed.
			<-time.After(delayTime)
			eventBus.Publish(topic)
			eventBus.Publish(topic)
			// Delay till the topicsMap are published.
			<-time.After(delayTime)

			assert.Equal(t, len(consumers), len(eventBus.dispatcher.(*dispatcher).getMatchedConsumers(topic)), "Event bus should have a matching number of consumers.")

			for _, consumer := range consumers {
				logMessage := fmt.Sprintf("Topic %s subscription is announced by %s.OnSubscribe", topic, consumer)
				assert.Equal(t, logMessage, eventBus.logger.(*fakeEventLogger).get(), "Subscription messages.")
			}

			assert.Equal(t, testCase.expectedLogMessage, eventBus.logger.(*fakeEventLogger).get(), "Event bus should have a log message for the first publish.")
			assert.Equal(t, testCase.expectedLogMessage, eventBus.logger.(*fakeEventLogger).get(), "Event bus should have a log message for the second publish.")

			for key, consumer := range consumers {
				assert.Equal(t, testCase.expectedTopic, consumer.ReadMessage(), fmt.Sprintf("Consumer %d should have received the first message.", key))
				assert.Equal(t, testCase.expectedTopic, consumer.ReadMessage(), fmt.Sprintf("Consumer %d should have received the second message.", key))
			}
		})
	}
}

func TestPublishAfter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    Descriptor
		expected float64
	}{
		{
			"Publish should be done after the duration is passed.",
			newTestDescriptor(),
			0.1,
		},
	}

	fakeConsumer := newFakeConsumer()

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			topic := testCase.input
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			eventBus := newEventBus(ctx, 1)

			err := eventBus.Subscribe(fakeConsumer, topic)

			if err != nil {
				t.Error(err)
			}

			currentTime := time.Now()
			eventBus.PublishAfter(ctx, 100*time.Millisecond, topic)

			assert.Equal(t, topic, fakeConsumer.ReadMessage(), testCase.id)
			assert.True(t, time.Now().Sub(currentTime).Seconds() >= testCase.expected, testCase.id)
		})
	}
}

func TestSchedulePublish(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Long test.")
	}

	testCases := []struct {
		id                      string
		input                   map[string]interface{}
		expectedPublishingTimes int
		isErrorExpected         bool
		expected                string
	}{
		{
			"Should publish based on schedule till timeout is reached.",
			map[string]interface{}{
				"topic":           newTestDescriptor(),
				"tickingDuration": time.Duration(100) * time.Millisecond,
				"waitingDuration": time.Duration(888) * time.Millisecond,
			},
			8,
			true,
			"context deadline exceeded",
		},
		{
			"Should not publish at all as it would be cancelled intentionally.",
			map[string]interface{}{
				"topic":           newTestDescriptor(),
				"tickingDuration": time.Duration(100) * time.Microsecond,
				"waitingDuration": time.Duration(1) * time.Millisecond,
			},
			0,
			true,
			"context canceled",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			topic := testCase.input["topic"].(Descriptor)
			tickingDuration := testCase.input["tickingDuration"].(time.Duration)
			waitingDuration := testCase.input["waitingDuration"].(time.Duration)

			ctx, cancel := context.WithTimeout(context.Background(), waitingDuration)
			defer cancel()
			eventBus := newEventBus(ctx, testBufferSize)

			fakeConsumer := newFakeConsumer()
			err := eventBus.Subscribe(fakeConsumer, topic)
			if err != nil {
				t.Error(err)
			}

			ticker := time.NewTicker(tickingDuration)
			eventBus.SchedulePublish(ctx, ticker, topic)

			if testCase.expectedPublishingTimes != 0 {
				<-time.After(waitingDuration)
			} else {
				cancel()
			}

			if testCase.isErrorExpected {
				err = <-eventBus.errorQueue.(*fakeErrorQueue).errChan
				assert.Equal(t, testCase.expected, err.Error(), testCase.id)
			}

			for ind := 0; ind < testCase.expectedPublishingTimes; ind++ {
				assert.Equal(t, topic, fakeConsumer.ReadMessage(), testCase.id)
			}
		})
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()

	var (
		fakeConsumerEntity *fakeConsumer
		nilTopic           Descriptor
	)
	doubleSubscriptionTopic := newTestDescriptor()

	testCases := []struct {
		id                 string
		input              map[string]interface{}
		expected           error
		isErrorExpected    bool
		expectedLogMessage string
	}{
		{
			"Consumer should not be nil.",
			map[string]interface{}{
				"consumers": []Consumer{
					fakeConsumerEntity,
				},
				"topicsMap": []Descriptor{},
			},
			errors.E(errors.Operation("Bus.Subscribe"), errors.Invalid),
			true,
			"",
		},
		{
			"Topics to subscribe at should be valid.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
				},
				"topicsMap": []Descriptor{},
			},
			errors.E(errors.Operation("Bus.Subscribe"), errors.MinLength),
			true,
			"",
		},
		{
			"Subscription for nil topics should not be possible.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
				},
				"topicsMap": []Descriptor{
					nilTopic,
				},
			},
			errors.E(errors.Operation("Bus.subscribe"), errors.Invalid),
			true,
			"",
		},
		{
			"Double subscription for the same topic should not be possible.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
				},
				"topicsMap": []Descriptor{
					doubleSubscriptionTopic,
					doubleSubscriptionTopic,
				},
			},
			errors.E(errors.Operation("Bus.subscribe"), errors.Exist, errors.Errorf("testTopic@v1")),
			true,
			"",
		},
		{
			"Subscription for valid topicsMap should be done correctly.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
				},
				"topicsMap": []Descriptor{
					newTestDescriptor(),
					newTestDescriptor(),
				},
			},
			nil,
			false,
			"Topic testTopic@v1 subscription is announced by fakeConsumer.OnSubscribe",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			consumers := testCase.input["consumers"].([]Consumer)
			topics := testCase.input["topicsMap"].([]Descriptor)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			eventBus := newEventBus(ctx, 1)

			for _, consumer := range consumers {
				err := eventBus.Subscribe(consumer, topics...)
				// Delay till the topics are subscribed.
				<-time.After(delayTime)

				if !testCase.isErrorExpected {
					assert.Equal(t, testCase.expected, err, testCase.id)
					assert.Equal(t, true, consumer.(*fakeConsumer).getOnSubscribeTriggerValue(), testCase.id)
					assert.Equal(t, testCase.expectedLogMessage, eventBus.logger.(*fakeEventLogger).get(), "Event bus should have a log message for the subscribe action.")
					assert.Equal(t, len(topics), eventBus.Length(), testCase.id)

					continue
				}

				if err != nil {
					assert.Equal(t, testCase.expected, err, testCase.id)
				} else {
					assert.Equal(t, testCase.expected, eventBus.errorQueue.(*fakeErrorQueue).getError(), testCase.id)
				}
			}
		})
	}
}

func TestSubscribeToAll(t *testing.T) {
	t.Parallel()

	var (
		fakeConsumerEntity *fakeConsumer
	)

	testCases := []struct {
		id              string
		input           map[string]interface{}
		expected        error
		isErrorExpected bool
	}{
		{
			"Consumer should not be nil.",
			map[string]interface{}{
				"consumers": []Consumer{
					fakeConsumerEntity,
				},
				"topicsMap": []Descriptor{},
			},
			errors.E(errors.Operation("Bus.SubscribeToAll"), errors.Invalid),
			true,
		},
		{
			"Subscription for valid topicsMap should be done correctly.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
				},
				"topicsMap": []Descriptor{
					newTestDescriptor(),
				},
			},
			nil,
			false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			consumers := testCase.input["consumers"].([]Consumer)
			topics := testCase.input["topicsMap"].([]Descriptor)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			eventBus := newEventBus(ctx, 1)

			for _, consumer := range consumers {
				err := eventBus.SubscribeToAll(consumer)
				// Delay till the topicsMap are subscribed.
				<-time.After(delayTime)
				for _, topic := range topics {
					eventBus.Publish(topic)
				}

				if !testCase.isErrorExpected {
					assert.Equal(t, testCase.expected, err, testCase.id)
					assert.Equal(t, true, consumer.(*fakeConsumer).getOnSubscribeTriggerValue(), testCase.id)
					assert.Equal(t, len(topics), eventBus.Length(), testCase.id)

					continue
				}

				if err != nil {
					assert.Equal(t, testCase.expected, err, testCase.id)
				} else {
					assert.Equal(t, testCase.expected, eventBus.errorQueue.(*fakeErrorQueue).getError(), testCase.id)
				}
			}
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()

	notSubscribedTopic := newTestDescriptor()
	nonExistentTopic := newTestDescriptor()
	subscribedTopic := newTestDescriptor()

	firstConsumer := newIdentifiableFakeConsumer("firstConsumer")
	secondConsumer := newIdentifiableFakeConsumer("secondConsumer")

	testCases := []struct {
		id                     string
		input                  map[string]interface{}
		expected               interface{}
		isErrorExpected        bool
		expectedReportedErrors int
	}{
		{
			"Consumer should fail to unsubscribe when there are no topics.",
			map[string]interface{}{
				"topicsConsumersMap": map[Descriptor][]Consumer{
					notSubscribedTopic: {},
					subscribedTopic: {
						newFakeConsumer(),
						newFakeConsumer(),
					},
				},
				"topicsMap": []Descriptor{},
			},
			errors.E(errors.Operation("Bus.Unsubscribe"), errors.MinLength),
			true,
			0,
		},
		{
			"Consumer should fail to unsubscribe from not previously subscribed topics.",
			map[string]interface{}{
				"topicsConsumersMap": map[Descriptor][]Consumer{
					notSubscribedTopic: {},
					subscribedTopic: {
						firstConsumer,
						secondConsumer,
					},
				},
				"topicsMap": []Descriptor{nonExistentTopic},
			},
			[]error{
				errors.E(errors.Operation("Bus.unsubscribe"), errors.NotFound, errors.Errorf(" for consumer %#v to %s", firstConsumer, subscribedTopic)),
				errors.E(errors.Operation("Bus.unsubscribe"), errors.NotFound, errors.Errorf(" for consumer %#v to %s", secondConsumer, subscribedTopic)),
			},
			true,
			2,
		},
		{
			"Should unsubscribe correctly.",
			map[string]interface{}{
				"topicsConsumersMap": map[Descriptor][]Consumer{
					subscribedTopic: {
						newFakeConsumer(),
						newFakeConsumer(),
					},
				},
				"topicsMap": []Descriptor{subscribedTopic},
			},
			0,
			false,
			0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			topicConsumers := testCase.input["topicsConsumersMap"].(map[Descriptor][]Consumer)
			topics := testCase.input["topicsMap"].([]Descriptor)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			eventBus := newEventBus(ctx, 1)

			allConsumers := make([]Consumer, 0)

			for topic, consumers := range topicConsumers {
				for _, consumer := range consumers {
					allConsumers = append(allConsumers, consumer)

					err := eventBus.Subscribe(consumer, topic)

					if err != nil {
						t.Error(err)
					}
				}
			}
			// Delay till subscription happens.
			<-time.After(delayTime)

			for key, consumer := range allConsumers {
				err := eventBus.Unsubscribe(consumer, topics...)
				if !testCase.isErrorExpected {
					continue
				}

				if err != nil {
					assert.Equal(t, testCase.expected, err, testCase.id)
				} else {
					expectedError := testCase.expected.([]error)[key].(*errors.Error)
					gotError := eventBus.errorQueue.(*fakeErrorQueue).getError()
					assert.True(t, errors.Is(expectedError.Kind, gotError), testCase.id)
					assert.Equal(t, expectedError, gotError, testCase.id)

					testCase.expectedReportedErrors--
				}
			}

			if testCase.isErrorExpected {
				assert.Equal(t, 0, testCase.expectedReportedErrors, testCase.id)

				return
			}

			// Delay till unsubscription happens.
			<-time.After(delayTime)
			assert.Equal(t, testCase.expected.(int), eventBus.Length(), testCase.id)
		})
	}
}

func TestUnregister(t *testing.T) {
	t.Parallel()

	firstTopic := newNamedTestDescriptor("firstTopic")
	secondTopic := newNamedTestDescriptor("secondTopic")
	thirdTopic := newNamedTestDescriptor("thirdTopic")

	testCases := []struct {
		id                 string
		input              map[string]interface{}
		expected           interface{}
		expectedLogMessage string
		isErrorExpected    bool
	}{
		{
			"At least one topic should be passed.",
			map[string]interface{}{
				"topicsConsumersMap": map[Descriptor][]Consumer{},
				"topicsMap":          []Descriptor{},
			},
			errors.E(errors.Operation("bus.Unregister"), errors.MinLength),
			"",
			true,
		},
		{
			"Expected topics should be unregistered.",
			map[string]interface{}{
				"topicsConsumersMap": map[Descriptor][]Consumer{
					firstTopic: {
						newFakeConsumer(),
						newFakeConsumer(),
					},
					secondTopic: {
						newFakeConsumer(),
					},
					thirdTopic: {
						newFakeConsumer(),
					},
				},
				"topicsMap": []Descriptor{firstTopic, secondTopic},
			},
			1,
			"Unregistering topicsMap [firstTopic@v1 secondTopic@v1]",
			false,
		},
		{
			"All topics should be unregistered.",
			map[string]interface{}{
				"topicsConsumersMap": map[Descriptor][]Consumer{
					firstTopic: {
						newFakeConsumer(),
						newFakeConsumer(),
					},
					secondTopic: {
						newFakeConsumer(),
					},
					thirdTopic: {
						newFakeConsumer(),
						newFakeConsumer(),
					},
				},
				"topicsMap": []Descriptor{firstTopic, secondTopic, thirdTopic},
			},
			0,
			"Unregistering topicsMap [firstTopic@v1 secondTopic@v1 thirdTopic@v1]",
			false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			topicConsumers := testCase.input["topicsConsumersMap"].(map[Descriptor][]Consumer)
			topics := testCase.input["topicsMap"].([]Descriptor)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			eventBus := newEventBus(ctx, testBufferSize)

			subscribe(t, eventBus, topicConsumers)

			err := eventBus.Unregister(topics...)

			if !testCase.isErrorExpected {
				// Delay till the topicsMap are unregistered, in case of no error.
				<-time.After(delayTime)

				for {
					logMessage := eventBus.logger.(*fakeEventLogger).get()
					// Ignoring other log messages.
					if !strings.HasPrefix(logMessage, "Unregistering") {
						continue
					}

					assert.Equal(t, testCase.expectedLogMessage, logMessage, "Event bus should have a log message for the unregister action.")
					break
				}
				assert.Equal(t, testCase.expected.(int), eventBus.Length(), "Event bus length should match the expected length.")

				return
			}

			if err != nil {
				assert.Equal(t, testCase.expected, err, testCase.id)

				return
			}

			assert.Equal(t, testCase.expected, eventBus.errorQueue.(*fakeErrorQueue).getError(), testCase.id)
		})
	}
}

func getFakeConsumers(size int) []Consumer {
	var consumers []Consumer

	for index := 0; index < size; index++ {
		consumers = append(consumers, newFakeConsumer())
	}

	return consumers
}

func subscribe(t *testing.T, eventBus *bus, topicConsumers map[Descriptor][]Consumer) {
	t.Helper()
	for topic, consumers := range topicConsumers {
		for _, consumer := range consumers {
			err := eventBus.Subscribe(consumer, topic)

			if err != nil {
				t.Error(err)
			}
		}
	}

	// Delay till the topicsMap are subscribed.
	<-time.After(delayTime)
}
