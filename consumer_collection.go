package eventually

import "sync"

type (
	// ConsumerCollection defines consumers collection operations.
	ConsumerCollection interface {
		// Append a consumer to the ConsumerCollection.
		Append(...Consumer)
		// Delete a consumer from the ConsumerCollection.
		Delete(...Consumer)
		// Length returns the length of the ConsumerCollection.
		Length() int
		// Exists check if consumers do exist in the ConsumerCollection.
		Exists(Consumer) bool
		// Iterator iterates over the consumers in a ConsumerCollection.
		Iterator() <-chan Consumer
	}

	// consumerCollection contains a list of consumers of a topic.
	consumerCollection struct {
		mux       sync.Mutex
		consumers map[Consumer]struct{}
	}
)

// newConsumerCollection creates a ConsumerCollection.
func newConsumerCollection() ConsumerCollection {
	return &consumerCollection{
		consumers: make(map[Consumer]struct{}),
	}
}

func (cc *consumerCollection) Append(consumers ...Consumer) {
	cc.mux.Lock()
	for _, consumer := range consumers {
		cc.consumers[consumer] = struct{}{}
	}
	cc.mux.Unlock()
}

func (cc *consumerCollection) Exists(consumer Consumer) bool {
	cc.mux.Lock()
	_, ok := cc.consumers[consumer]
	cc.mux.Unlock()

	return ok
}

func (cc *consumerCollection) Delete(consumers ...Consumer) {
	cc.mux.Lock()
	for _, consumer := range consumers {
		delete(cc.consumers, consumer)
	}
	cc.mux.Unlock()
}

func (cc *consumerCollection) Length() int {
	cc.mux.Lock()
	length := len(cc.consumers)
	cc.mux.Unlock()

	return length
}

func (cc *consumerCollection) Iterator() <-chan Consumer {
	c := make(chan Consumer, cc.Length())

	cc.mux.Lock()
	for consumer := range cc.consumers {
		c <- consumer
	}
	cc.mux.Unlock()

	close(c)

	return c
}
