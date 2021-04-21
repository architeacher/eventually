package eventually

import (
	"flag"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"reflect"
	"testing"
	"time"
)

type (
	fakeInbox struct {
		messagesChan       chan Descriptor
		isSignoutTriggered bool
	}

	testConsumer struct {
		id      UUID
		mailBox Inbox
	}
)

const (
	defaultTimeout = 8 * time.Second
)

var timeMultiplier = flag.Int("timeMultiplier", 1, "time multiplier")

func newFakeInbox(space uint) Inbox {
	return &fakeInbox{
		messagesChan: make(chan Descriptor, space),
	}
}

func (f *fakeInbox) Receive(topic Descriptor) {
	select {
	case f.messagesChan <- topic:
		log.Printf("Received topic %s\n", topic)
	//Drop any message that exceeds the mail box space.
	default:
	}
}

func (f *fakeInbox) Read() Descriptor {
	timeout := defaultTimeout * time.Duration(*timeMultiplier)

	select {
	case topic := <-f.messagesChan:
		return topic
	// Making tests fail fast by returning an empty message.
	case <-time.After(timeout):
		return NewDescriptor("nullEvent", new(nullPayload), 1)
	}
}

func (f *fakeInbox) Signout() {
	f.isSignoutTriggered = true
	close(f.messagesChan)
}

// newTestConsumer creates a universal consumer.
func newTestConsumer(mailBox Inbox) Consumer {
	return &testConsumer{
		id:      NewUUID(),
		mailBox: mailBox,
	}
}

func (tc *testConsumer) AggregateID() UUID {
	return tc.id
}

func (tc *testConsumer) MatchCriteria() Match {
	return func(topic Descriptor) bool {
		return true
	}
}

func (tc *testConsumer) Drop(descriptor Descriptor) {
	tc.mailBox.Receive(descriptor)
}

func (tc *testConsumer) ReadMessage() Descriptor {
	return tc.mailBox.Read()
}

func (tc *testConsumer) OnSubscribe(topic Descriptor, callback Notification) {
	callback(topic, tc)
}

func (tc *testConsumer) OnUnsubscribe(topic Descriptor, callback Notification) {
	callback(topic, tc)
}

func (tc *testConsumer) Signout() {
	tc.mailBox.Signout()
}

func (tc *testConsumer) GoString() string {
	return fmt.Sprintf("%s[%s]", tc.String(), tc.id)
}

func (tc *testConsumer) String() string {
	return reflect.TypeOf(tc).Elem().Name()
}

func TestShouldMatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    Consumer
		expected bool
	}{
		{
			id:       "Should be always matching for universal consumer",
			input:    newTestConsumer(newFakeInbox(5)),
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			testTopic := newTestDescriptor()
			match := testCase.input.MatchCriteria()
			assert.Equal(t, testCase.expected, match(testTopic), testCase.id)
		})
	}
}

func TestDropReadMessage(t *testing.T) {
	t.Parallel()

	topic := newTestDescriptor()
	consumer := newFakeConsumer()

	testCases := []struct {
		id       string
		input    Descriptor
		expected Descriptor
	}{
		{
			id:       "Should drop and read message.",
			input:    topic,
			expected: topic,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			topic := testCase.input.(Descriptor)
			consumer.Drop(topic)

			assert.Equal(t, testCase.expected, consumer.ReadMessage(), testCase.id)
		})
	}
}

func TestShouldCallOnSubscribeNotification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    Consumer
		expected bool
	}{
		{
			id:       "Should call on subscribe notification.",
			input:    newTestConsumer(newFakeInbox(5)),
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumer := testCase.input.(Consumer)
			testChan := make(chan bool, 1)
			consumer.OnSubscribe(newTestDescriptor(), func(topic Descriptor, consumer Consumer) {
				testChan <- true
			})
			assert.Equal(t, testCase.expected, <-testChan)
		})
	}
}

func TestShouldCallOnUnSubscribeNotification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    Consumer
		expected bool
	}{
		{
			id:       "Should call on unsubscribe notification.",
			input:    newTestConsumer(newFakeInbox(5)),
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumer := testCase.input.(Consumer)
			testChan := make(chan bool, 1)
			consumer.OnUnsubscribe(newTestDescriptor(), func(topic Descriptor, consumer Consumer) {
				testChan <- true
			})
			assert.Equal(t, testCase.expected, <-testChan)
		})
	}
}

func TestShouldSignoutInbox(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    Consumer
		expected bool
	}{
		{
			id:       "Should signout the inbox",
			input:    newFakeConsumer(),
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumer := testCase.input.(*fakeConsumer)
			consumer.Signout()
			assert.Equal(t, testCase.expected, consumer.mailBox.(*fakeInbox).isSignoutTriggered)
		})
	}
}
