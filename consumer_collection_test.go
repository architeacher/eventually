package eventually

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAppendToCollection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected int
	}{
		{
			"Should append consumer to the collection successfully.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			3,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumers := testCase.input["consumers"].([]Consumer)

			consumersCol := newConsumerCollection()

			for _, consumer := range consumers {
				consumersCol.Append(consumer)
			}

			assert.Equal(t, testCase.expected, consumersCol.Length(), testCase.id)
		})
	}
}

func TestExistsInCollection(t *testing.T) {
	t.Parallel()

	consumers := []Consumer{
		newFakeConsumer(),
		newFakeConsumer(),
		newFakeConsumer(),
	}

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected bool
	}{
		{
			"Should verify that consumer exists in the collection.",
			map[string]interface{}{
				"consumers":     consumers,
				"checkConsumer": consumers[0],
			},
			true,
		},
		{
			"Should verify that consumer does not exist in the collection.",
			map[string]interface{}{
				"consumers":     consumers,
				"checkConsumer": newFakeConsumer(),
			},
			false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumers := testCase.input["consumers"].([]Consumer)
			checkConsumer := testCase.input["checkConsumer"].(Consumer)

			consumersCol := newConsumerCollection()
			consumersCol.Append(consumers...)
			assert.Equal(t, testCase.expected, consumersCol.Exists(checkConsumer), testCase.id)
		})
	}
}

func TestDeleteFromCollection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected int
	}{
		{
			"Should delete consumers from the collection successfully.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumers := testCase.input["consumers"].([]Consumer)

			consumersCol := newConsumerCollection()
			consumersCol.Append(consumers...)
			assert.Equal(t, len(consumers), consumersCol.Length(), testCase.id)

			consumersCol.Delete(consumers...)
			assert.Equal(t, testCase.expected, consumersCol.Length(), testCase.id)
		})
	}
}

func TestLengthOfCollection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected int
	}{
		{
			"Should get the correct length of the collection.",
			map[string]interface{}{
				"consumers": []Consumer{
					newFakeConsumer(),
					newFakeConsumer(),
					newFakeConsumer(),
				},
			},
			3,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumers := testCase.input["consumers"].([]Consumer)

			consumersCol := newConsumerCollection()
			consumersCol.Append(consumers...)
			assert.Equal(t, testCase.expected, consumersCol.Length(), testCase.id)

			consumersCol.Delete(consumers...)
			assert.Equal(t, 0, consumersCol.Length(), testCase.id)
		})
	}
}

func TestIteratorOfCollection(t *testing.T) {
	t.Parallel()

	consumers := [...]*fakeConsumer{
		newFakeConsumer(),
		newFakeConsumer(),
		newFakeConsumer(),
	}

	consumers[0].id = "firstConsumer"
	consumers[1].id = "secondConsumer"
	consumers[2].id = "thirdConsumer"

	testCases := []struct {
		id       string
		input    map[string]interface{}
		expected int
	}{
		{
			"Should iterate over the collection.",
			map[string]interface{}{
				"consumers": consumers,
			},
			3,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			consumers := testCase.input["consumers"].([3]*fakeConsumer)
			consumersMap := make(map[*fakeConsumer]bool, 3)

			consumersCol := newConsumerCollection()
			for _, consumer := range consumers {
				consumersCol.Append(consumer)
				consumersMap[consumer] = true
			}

			index := 0
			for consumer := range consumersCol.Iterator() {
				assert.Equal(t, true, consumersMap[consumer.(*fakeConsumer)], testCase.id)
				index++
			}

			assert.Equal(t, testCase.expected, index, testCase.id)
		})
	}
}

func BenchmarkAppendIterateOfCollection(b *testing.B) {
	consumers := getFakeConsumers(1000)
	consumersCol := newConsumerCollection()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			consumersCol.Append(consumers...)
			for consumer := range consumersCol.Iterator() {
				consumer.MatchCriteria()
			}
		}
	})
}
