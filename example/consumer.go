package main

import (
	"fmt"
	"github.com/ahmedkamals/eventually"
	"reflect"
)

type (
	exampleConsumer struct {
		id      eventually.UUID
		mailBox eventually.Inbox
	}
)

func newExampleConsumer(ID eventually.UUID, mailBox eventually.Inbox) eventually.Consumer {
	return &exampleConsumer{
		id:      ID,
		mailBox: mailBox,
	}
}

func (ec *exampleConsumer) AggregateID() eventually.UUID {
	return ec.id
}

func (ec *exampleConsumer) MatchCriteria() eventually.Match {
	return func(topic eventually.Descriptor) bool {
		return true
	}
}

func (ec *exampleConsumer) Drop(descriptor eventually.Descriptor) {
	ec.mailBox.Receive(descriptor)
}

func (ec *exampleConsumer) ReadMessage() eventually.Descriptor {
	return ec.mailBox.Read()
}

func (ec *exampleConsumer) OnSubscribe(topic eventually.Descriptor, callback eventually.Notification) {
	callback(topic, ec)
}

func (ec *exampleConsumer) OnUnsubscribe(topic eventually.Descriptor, callback eventually.Notification) {
	callback(topic, ec)
}

func (ec *exampleConsumer) Signout() {
	ec.mailBox.Signout()
}

func (ec *exampleConsumer) GoString() string {
	return fmt.Sprintf("%s[%s]", ec.String(), ec.id)
}

func (ec *exampleConsumer) String() string {
	return reflect.TypeOf(ec).Elem().Name()
}
