package eventually

import (
	"sync"
)

type (
	// EventStore for the topicsMap.
	EventStore interface {
		Load(UUID) (ConsumerCollection, bool)
		Store(Descriptor, ...Consumer) bool
		DeleteTopic(...Descriptor)
		Length() int
		Remove(Descriptor, ...Consumer) bool
		IsSubscribed(Descriptor, Consumer) bool
		Topics() []Descriptor
	}

	eventStore struct {
		sync.RWMutex
		topicsConsumersMap map[UUID]ConsumerCollection
		topicsMap          map[Descriptor]struct{}
		topicsMapMux       sync.Mutex
	}
)

// NewEventStore creates a new EventStore.
func NewEventStore() EventStore {
	return &eventStore{
		topicsConsumersMap: make(map[UUID]ConsumerCollection),
		topicsMap:          make(map[Descriptor]struct{}),
	}
}

func (es *eventStore) Load(aggregateID UUID) (ConsumerCollection, bool) {
	es.RLock()
	consumersCol, ok := es.topicsConsumersMap[aggregateID]
	es.RUnlock()

	return consumersCol, ok
}

func (es *eventStore) Store(topic Descriptor, consumers ...Consumer) bool {
	if len(consumers) == 0 {
		return false
	}

	consumersCol, ok := es.Load(topic.AggregateID())
	es.Lock()
	if !ok {
		consumersCol = newConsumerCollection()
		es.appendTopic(topic)
	}
	consumersCol.Append(consumers...)
	es.Unlock()

	es.updateTopicsMap(topic.AggregateID(), consumersCol)

	return true
}

func (es *eventStore) DeleteTopic(topics ...Descriptor) {
	es.Lock()
	defer es.Unlock()

	for _, topic := range topics {
		delete(es.topicsConsumersMap, topic.AggregateID())
		es.removeTopic(topic)
	}
}

func (es *eventStore) Length() int {
	es.Lock()
	defer es.Unlock()

	return len(es.topicsConsumersMap)
}

func (es *eventStore) Remove(topic Descriptor, consumers ...Consumer) bool {
	if len(consumers) == 0 {
		return false
	}

	consumersCol, ok := es.Load(topic.AggregateID())
	if !ok {
		return false
	}

	consumersCol.(ConsumerCollection).Delete(consumers...)
	if consumersCol.Length() == 0 {
		es.DeleteTopic(topic)

		return true
	}

	es.updateTopicsMap(topic.AggregateID(), consumersCol)

	return true
}

func (es *eventStore) IsSubscribed(topic Descriptor, consumer Consumer) bool {
	consumersCol, ok := es.Load(topic.AggregateID())

	if ok {
		ok = consumersCol.Exists(consumer)
	}

	return ok
}

func (es *eventStore) updateTopicsMap(aggregateID UUID, consumersCol ConsumerCollection) {
	es.Lock()
	defer es.Unlock()

	es.topicsConsumersMap[aggregateID] = consumersCol
}

func (es *eventStore) Topics() []Descriptor {
	es.Lock()
	defer es.Unlock()

	topics := make([]Descriptor, len(es.topicsMap))
	index := 0

	for topic := range es.topicsMap {
		topics[index] = topic
		index++
	}

	return topics
}

func (es *eventStore) appendTopic(topics ...Descriptor) {
	es.topicsMapMux.Lock()
	defer es.topicsMapMux.Unlock()

	for _, topic := range topics {
		es.topicsMap[topic] = struct{}{}
	}
}

func (es *eventStore) removeTopic(topics ...Descriptor) {
	es.topicsMapMux.Lock()
	defer es.topicsMapMux.Unlock()

	for _, topic := range topics {
		delete(es.topicsMap, topic)
	}
}
