package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ahmedkamals/colorize"
	"github.com/ahmedkamals/eventually"
)

type (
	eventLogger struct {
		logChan chan string
	}

	errorQueue struct {
		errChan chan error
	}

	universalPayload struct{}
)

const (
	// MailboxDefaultSize defines size for the consumer mail box.
	MailboxDefaultSize = 10
	universal          = "universal"
)

var (
	colorized = colorize.NewColorable(os.Stdout)
)

func newEventLogger(logChan chan string) eventually.Logger {
	return &eventLogger{
		logChan: logChan,
	}
}

func (e *eventLogger) Log(message string) {
	select {
	case e.logChan <- message:
	// Drop any log message that exceeds the log queue size.
	default:
	}
}

func newErrorQueue(errChan chan error) eventually.ErrorQueue {
	return &errorQueue{
		errChan: errChan,
	}
}

func (e *errorQueue) Report(err error) {
	select {
	case e.errChan <- err:
	// Drop any error message that exceeds the error queue size.
	default:
	}
}

func main() {
	logChan := make(chan string, 10)
	errChan := make(chan error, 10)

	go monitorLogMessages(logChan)
	go monitorErrors(errChan)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	eventBus := eventually.NewBus(eventually.NewEventStore(), newEventLogger(logChan), newErrorQueue(errChan), 100)
	eventBus.Run(ctx)

	topics := map[string]eventually.Descriptor{
		universal:         eventually.NewDescriptor(universal, new(universalPayload), 1),
		"example":         eventually.NewDescriptor("example", "examplePayload", 1),
		"anotherExample":  eventually.NewDescriptor("anotherExample", "anotherExamplePayload", 2),
		"publishAfter":    eventually.NewDescriptor("publishAfter", "publishAfterPayload", 1),
		"schedulePublish": eventually.NewDescriptor("schedulePublish", "schedulePublishPayload", 1),
	}

	topicsConsumersMap := map[eventually.Descriptor][]eventually.Consumer{
		topics[universal]: {
			newExampleConsumer(eventually.NewUUID(), eventually.NewInbox(MailboxDefaultSize)),
		},
		topics["example"]: {
			newExampleConsumer(eventually.NewUUID(), eventually.NewInbox(MailboxDefaultSize)),
			newExampleConsumer(eventually.NewUUID(), eventually.NewInbox(MailboxDefaultSize)),
			newExampleConsumer(eventually.NewUUID(), eventually.NewInbox(MailboxDefaultSize)),
		},
		topics["anotherExample"]: {
			newExampleConsumer(eventually.NewUUID(), eventually.NewInbox(MailboxDefaultSize)),
		},
		topics["publishAfter"]: {
			newExampleConsumer(eventually.NewUUID(), eventually.NewInbox(MailboxDefaultSize)),
		},
		topics["schedulePublish"]: {
			newExampleConsumer(eventually.NewUUID(), eventually.NewInbox(MailboxDefaultSize)),
		},
	}

	subscribe(eventBus, topicsConsumersMap, errChan)

	eventBus.Publish(topics["example"], topics["anotherExample"])
	eventBus.PublishAfter(ctx, time.Second, topics["publishAfter"])
	eventBus.SchedulePublish(ctx, time.NewTicker(800*time.Millisecond), topics["schedulePublish"])

	<-time.After(88 * time.Millisecond)
	fmt.Printf("%s, Subscribed events[%d]: %#v\n", colorized.White("After publishing"), eventBus.Length(), eventBus.Topics())

	eventBus.Unsubscribe(topicsConsumersMap[topics["example"]][0], topics["example"])
	<-time.After(88 * time.Millisecond)
	fmt.Printf("%s, Subscribed events[%d]: %#v\n", colorized.Magenta("After unsubscribing"), eventBus.Length(), eventBus.Topics())

	eventBus.Unregister(topics["anotherExample"])
	<-time.After(888 * time.Millisecond)
	fmt.Printf("%s, Subscribed events[%d]: %#v\n", colorized.Yellow("After unregistering"), eventBus.Length(), eventBus.Topics())

	// Delay till all topics are published.
	<-time.After(3 * time.Second)
}

func subscribe(eventBus eventually.Bus, topicsConsumersMap map[eventually.Descriptor][]eventually.Consumer, errChan chan error) {

	for topic, consumers := range topicsConsumersMap {
		for _, consumer := range consumers {
			go func(consumer eventually.Consumer) {
				for {
					message := consumer.ReadMessage()
					if message == nil {
						continue
					}
					fmt.Printf(
						"%s[%s] Inbox: got message %s\n",
						colorized.Green(consumer.String()),
						consumer.AggregateID(),
						colorized.Orange(message.Payload().(string)),
					)
				}
			}(consumer)

			switch topic.Name() {
			case universal:
				errChan <- eventBus.SubscribeToAll(consumer)

			default:
				errChan <- eventBus.Subscribe(consumer, topic)
			}
		}
	}
	// Delay till subscription happens.
	<-time.After(8 * time.Millisecond)
}

func monitorLogMessages(logChan chan string) {
	for message := range logChan {
		fmt.Println(colorized.Cyan(message))
	}
}

func monitorErrors(errChan chan error) {
	for err := range errChan {
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Println(colorized.Red(err.Error()))
		}
	}
}
