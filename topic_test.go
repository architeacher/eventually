package eventually

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type (
	nullPayload struct{}
)

func newTestDescriptor() Descriptor {
	return NewDescriptor("testTopic", "testPayload", 1)
}

func newUniversalTestDescriptor() Descriptor {
	return newUniversalDescriptor("testTopic", "testPayload", 1)
}

func newNamedTestDescriptor(category Category) Descriptor {
	return NewDescriptor(category, "testPayload", 1)
}

func TestAggregateID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    UUID
		expected UUID
	}{
		{
			"Should have a new aggregate id.",
			newTestDescriptor().AggregateID(),
			newTestDescriptor().AggregateID(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			assert.NotEqual(t, testCase.expected, testCase.input, testCase.id)
		})
	}
}

func TestName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    Descriptor
		expected Category
	}{
		{
			"Should evaluate name correctly.",
			newTestDescriptor(),
			Category("testTopic"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, testCase.input.Name(), testCase.id)
		})
	}
}

func TestGoString(t *testing.T) {
	t.Parallel()

	testDescriptor := newTestDescriptor()
	testDescriptor.(*descriptor).ID = "test"

	testCases := []struct {
		id       string
		input    Descriptor
		expected string
	}{
		{
			"Should return correct go string representation.",
			testDescriptor,
			"testTopic@v1[test]",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, fmt.Sprintf("%#v", testCase.input), testCase.id)
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id       string
		input    Descriptor
		expected string
	}{
		{
			"Should return correct string representation.",
			newTestDescriptor(),
			"testTopic@v1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, fmt.Sprintf("%s", testCase.input), testCase.id)
		})
	}
}

func TestSetHeader(t *testing.T) {
	t.Parallel()

	topic := newTestDescriptor()
	topic.(*descriptor).MetaData["test"] = "value"
	topic.SetHeader("test", "overridden-value")

	testCases := []struct {
		id       string
		input    string
		expected string
	}{
		{
			"Should set headers correctly.",
			"test",
			"overridden-value",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			metadata, ok := topic.GetHeader(testCase.input)

			if !ok {
				t.Error("Topic is missing")
			}
			assert.Equal(t, testCase.expected, metadata.(string), testCase.id)
		})
	}
}

func TestGetHeaders(t *testing.T) {
	t.Parallel()

	topic := newTestDescriptor()
	topic.(*descriptor).MetaData["test"] = "value"
	topic.(*descriptor).MetaData["key"] = "value"

	testCases := []struct {
		id       string
		input    Descriptor
		expected Metadata
	}{
		{
			"Should get headers correctly.",
			topic,
			Metadata{
				"test": "value",
				"key":  "value",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, testCase.input.GetHeaders(), testCase.id)
		})
	}
}

func TestMarshalling(t *testing.T) {
	t.Parallel()

	topic := newTestDescriptor()
	topic.SetHeader("test", "value")

	testCases := []struct {
		id    string
		input Descriptor
	}{
		{
			"Should marshall correctly.",
			topic,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			input := testCase.input

			marshaledData, err := input.Marshal()
			if err != nil {
				t.Error(err)
			}

			unMarshaledTopic := new(descriptor)
			err = unMarshaledTopic.Unmarshal(marshaledData)

			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, input.AggregateID(), unMarshaledTopic.AggregateID(), testCase.id)
			assert.Equal(t, input.Name(), unMarshaledTopic.Name(), testCase.id)
			assert.Equal(t, input.String(), unMarshaledTopic.String(), testCase.id)
			assert.Equal(t, input.Version(), unMarshaledTopic.Version(), testCase.id)
			assert.Equal(t, input.CreatedAt().Unix(), unMarshaledTopic.CreatedAt().Unix(), testCase.id)
			assert.Equal(t, input.GetHeaders(), unMarshaledTopic.GetHeaders(), testCase.id)
			assert.Equal(t, input.Payload(), unMarshaledTopic.Payload(), testCase.id)
		})
	}
}

func TestSerialization(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id    string
		input Descriptor
	}{
		{
			"Should serialize correctly.",
			newTestDescriptor(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			input := testCase.input

			serializedData, err := input.Serialize()
			if err != nil {
				t.Error(err)
			}

			deserializedTopic := new(descriptor)
			deserializedTopic.serializer = NewSerializer()
			err = deserializedTopic.Deserialize(serializedData)

			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, input.AggregateID(), deserializedTopic.AggregateID(), testCase.id)
			assert.Equal(t, input.Name(), deserializedTopic.Name(), testCase.id)
			assert.Equal(t, input.String(), deserializedTopic.String(), testCase.id)
			assert.Equal(t, input.Version(), deserializedTopic.Version(), testCase.id)
			assert.Equal(t, input.CreatedAt().Unix(), deserializedTopic.CreatedAt().Unix(), testCase.id)
			assert.Equal(t, input.GetHeaders(), deserializedTopic.GetHeaders(), testCase.id)
			assert.Equal(t, input.Payload().(string), deserializedTopic.Payload().(string), testCase.id)
		})
	}
}
