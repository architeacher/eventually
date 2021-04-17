package eventually

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func newDispatcher(bufferSize int) *dispatcher {
	logger := newFakeEventLogger(make(chan string, bufferSize))
	queue := newFakeErrorQueue(make(chan error, bufferSize))

	return NewDispatcher(NewEventStore(), logger, queue, bufferSize).(*dispatcher)
}

func TestShouldGetMatchedConsumers(t *testing.T) {
	t.Parallel()

	topic := newTestDescriptor()
	universalConsumer := newIdentifiableFakeConsumer("Universal consumer.")
	topicConsumers := []Consumer{
		newIdentifiableFakeConsumer("First topic consumer."),
		newIdentifiableFakeConsumer("Second topic consumer."),
		newIdentifiableFakeConsumer("Third topic consumer."),
	}

	topicConsumers[2].(*fakeConsumer).shouldMatch = false

	testCases := []struct {
		id                string
		input             map[Descriptor][]Consumer
		checkTopic        Descriptor
		expectedConsumers map[Consumer]struct{}
	}{
		{
			"Should get matched relevant topic and universal consumers.",
			map[Descriptor][]Consumer{
				newUniversalTestDescriptor(): {
					universalConsumer,
				},
				topic: topicConsumers,
				// Should not match these ones.
				newTestDescriptor(): {
					newIdentifiableFakeConsumer("Another topic consumer."),
				},
			},
			topic,
			map[Consumer]struct{}{
				universalConsumer: {},
				topicConsumers[0]: {},
				topicConsumers[1]: {},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			topicsConsumersMap := testCase.input
			checkTopic := testCase.checkTopic

			dispatcher := newDispatcher(testBufferSize)

			for topic, consumers := range topicsConsumersMap {
				dispatcher.eventStore.Store(topic, consumers...)
			}

			matchedConsumers := dispatcher.getMatchedConsumers(checkTopic)
			assert.Equal(t, len(topicsConsumersMap[checkTopic]), len(matchedConsumers), "Dispatcher should have a matching number of consumers.")

			for _, matchedConsumer := range matchedConsumers {
				_, ok := testCase.expectedConsumers[matchedConsumer]
				assert.Equal(t, true, ok, "Dispatcher should have the same consumers.")
			}
		})
	}
}
