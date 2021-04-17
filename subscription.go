package eventually

type (
	subscription interface {
		getConsumer() Consumer
		getTopics() []Descriptor
		isUniversal() bool
	}

	universalSubscription struct {
		consumer Consumer
	}

	normalSubscription struct {
		consumer Consumer
		topics   []Descriptor
	}
)

func newUniversalSubscription(consumer Consumer) subscription {
	return &universalSubscription{
		consumer: consumer,
	}
}

func (us *universalSubscription) getConsumer() Consumer {
	return us.consumer
}

func (us *universalSubscription) getTopics() []Descriptor {
	return []Descriptor{}
}

func (us *universalSubscription) isUniversal() bool {
	return true
}

func newNormalSubscription(consumer Consumer, topics ...Descriptor) subscription {
	return &normalSubscription{
		consumer: consumer,
		topics:   topics,
	}
}

func (ns *normalSubscription) getConsumer() Consumer {
	return ns.consumer
}

func (ns *normalSubscription) isUniversal() bool {
	return false
}

func (ns *normalSubscription) getTopics() []Descriptor {
	return ns.topics
}
