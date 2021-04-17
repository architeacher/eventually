package eventually

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadStore(t *testing.T) {
	t.Parallel()

	topic := newTestDescriptor()
	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected bool
	}{
		{
			"Should fail to store data.",
			map[string]interface{}{
				"saveTopic": newTestDescriptor(),
				"loadTopic": newTestDescriptor(),
				"consumers": []Consumer{},
			},
			false,
		},
		{
			"Should fail to load data.",
			map[string]interface{}{
				"saveTopic": newTestDescriptor(),
				"loadTopic": newTestDescriptor(),
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			false,
		},
		{
			"Should load and store data correctly.",
			map[string]interface{}{
				"saveTopic": topic,
				"loadTopic": topic,
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			saveTopic := testCase.input["saveTopic"].(Descriptor)
			loadTopic := testCase.input["loadTopic"].(Descriptor)
			consumers := testCase.input["consumers"].([]Consumer)

			store := NewEventStore()
			isStored := store.Store(saveTopic, consumers...)

			assert.Equal(t, testCase.expected, isStored, "Storing data.")
			consumerCol, ok := store.Load(loadTopic.AggregateID())

			assert.Equal(t, testCase.expected, ok, testCase.id)

			if !testCase.expected {
				assert.Equal(t, nil, consumerCol, testCase.id)

				return
			}

			for key, consumer := range consumers {
				assert.Equal(t, testCase.expected, consumerCol.Exists(consumer), fmt.Sprintf("Consumer %d should exists", key))
			}
			assert.Equal(t, len(consumers), consumerCol.Length(), "Loaded consumers should match the saved ones.")
		})
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()

	saveTopic := newTestDescriptor()
	checkTopic := newTestDescriptor()

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected bool
	}{
		{
			"Should do nothing when removing zero consumers.",
			map[string]interface{}{
				"saveTopic":  saveTopic,
				"checkTopic": checkTopic,
				"consumers":  []Consumer{},
			},
			false,
		},
		{
			"Should do nothing when removing non existing consumer.",
			map[string]interface{}{
				"saveTopic":  saveTopic,
				"checkTopic": checkTopic,
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			false,
		},
		{
			"Should remove an existing consumer correctly.",
			map[string]interface{}{
				"saveTopic":  saveTopic,
				"checkTopic": saveTopic,
				"consumers": []Consumer{
					newFakeConsumer(),
				},
			},
			true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			saveTopic := testCase.input["saveTopic"].(Descriptor)
			checkTopicType := testCase.input["checkTopic"].(Descriptor)
			consumers := testCase.input["consumers"].([]Consumer)

			store := NewEventStore()
			store.Store(saveTopic, consumers...)
			isRemoved := store.Remove(checkTopicType, consumers...)

			assert.Equal(t, testCase.expected, isRemoved, "Checking if consumers are removed.")

			consumerCol, ok := store.Load(saveTopic.AggregateID())
			if ok {
				for _, consumer := range consumers {
					assert.Equal(t, true, consumerCol.Exists(consumer), testCase.id)
				}
			} else {
				assert.Equal(t, 0, store.Length(), testCase.id)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected bool
	}{
		{
			"Should not fail to delete data if not exist.",
			map[string]interface{}{
				"saveTopic":   newTestDescriptor(),
				"deleteTopic": newTestDescriptor(),
				"consumers": []Consumer{
					newFakeConsumer(),
				},
			},
			false,
		},
		{
			"Should delete data correctly.",
			map[string]interface{}{
				"saveTopic":   newTestDescriptor(),
				"deleteTopic": newTestDescriptor(),
				"consumers": []Consumer{
					newFakeConsumer(),
				},
			},
			false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			saveTopic := testCase.input["saveTopic"].(Descriptor)
			deleteTopic := testCase.input["deleteTopic"].(Descriptor)
			consumers := testCase.input["consumers"].([]Consumer)

			store := NewEventStore()
			store.Store(saveTopic, consumers...)
			store.DeleteTopic(deleteTopic)
			consumerCol, ok := store.Load(deleteTopic.AggregateID())

			assert.Equal(t, testCase.expected, ok, testCase.id)

			if testCase.expected {
				assert.Equal(t, testCase.expected, consumerCol.Exists(consumers[0]), testCase.id)
				return
			}

			assert.Equal(t, nil, consumerCol, testCase.id)
		})
	}
}

func TestLength(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected int
	}{
		{
			"Should return zero if there are no topics.",
			map[string]interface{}{
				"topicsMap": []Descriptor{},
				"consumers": []Consumer{
					newFakeConsumer(),
				},
			},
			0,
		},
		{
			"Should return existing topics.",
			map[string]interface{}{
				"topicsMap": []Descriptor{
					newTestDescriptor(),
					newTestDescriptor(),
				},
				"consumers": []Consumer{
					newFakeConsumer(),
				},
			},
			2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			topics := testCase.input["topicsMap"].([]Descriptor)
			consumers := testCase.input["consumers"].([]Consumer)

			store := NewEventStore()
			for _, topic := range topics {
				store.Store(topic, consumers...)
			}

			assert.Equal(t, len(topics), store.Length(), testCase.id)

			for _, topic := range topics {
				store.DeleteTopic(topic)
			}

			assert.Equal(t, 0, store.Length(), testCase.id)
		})
	}
}

func TestIsSubscribed(t *testing.T) {
	t.Parallel()

	saveTopic := newTestDescriptor()
	checkTopic := newTestDescriptor()

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected bool
	}{
		{
			"Should return false when not subscribed to an saveTopic.",
			map[string]interface{}{
				"saveTopic":  saveTopic,
				"checkTopic": checkTopic,
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			false,
		},
		{
			"Should return false when subscribed to an saveTopic.",
			map[string]interface{}{
				"saveTopic":  saveTopic,
				"checkTopic": saveTopic,
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			saveTopic := testCase.input["saveTopic"].(Descriptor)
			checkTopic := testCase.input["checkTopic"].(Descriptor)
			consumers := testCase.input["consumers"].([]Consumer)

			store := NewEventStore()
			store.Store(saveTopic, consumers...)

			for _, consumer := range consumers {
				assert.Equal(t, testCase.expected, store.IsSubscribed(checkTopic, consumer), testCase.id)
			}
		})
	}
}

func TestShouldStoreLoadTopics(t *testing.T) {
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
				"topics": []Descriptor{},
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			[]Descriptor{},
		},
		{
			"Should return non duplicate existing topics.",
			map[string]interface{}{
				"topics": topics,
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			topics,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			topics := testCase.input["topics"].([]Descriptor)
			consumers := testCase.input["consumers"].([]Consumer)

			store := NewEventStore()
			for _, topic := range topics {
				store.Store(topic, consumers...)
			}

			assert.ElementsMatch(t, testCase.expected, store.Topics(), "should return the same topics")

			for _, topic := range topics {
				store.DeleteTopic(topic)
			}

			assert.Equal(t, 0, len(store.Topics()), "should have the same length")
		})
	}
}
