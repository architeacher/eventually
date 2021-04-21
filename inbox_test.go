package eventually

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadReceive(t *testing.T) {
	topics := [...]Descriptor{
		newTestDescriptor(),
		newTestDescriptor(),
		newTestDescriptor(),
	}
	inbox := NewInbox(uint(len(topics)))

	for _, topic := range topics {
		inbox.Receive(topic)
	}

	testCases := []struct {
		id       string
		input    Inbox
		expected [3]Descriptor
	}{
		{
			"Should receive and read messages normally.",
			inbox,
			topics,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			t.Parallel()

			for _, topic := range testCase.expected {
				assert.Equal(t, topic, testCase.input.Read(), testCase.id)
			}
		})
	}

	inbox.Signout()
}
