Eventually [![CircleCI](https://circleci.com/gh/ahmedkamals/eventually.svg?style=svg)](https://circleci.com/gh/ahmedkamals/eventually "Build Status")
==========

[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](LICENSE  "License")
[![release](https://img.shields.io/github/release/ahmedkamals/eventually.svg?style=flat-square)](https://github.com/ahmedkamals/eventually/releases/latest "Release")
[![Coverage Status](https://coveralls.io/repos/github/ahmedkamals/eventually/badge.svg?branch=master)](https://coveralls.io/github/ahmedkamals/eventually?branch=master  "Code Coverage")
[![codecov](https://codecov.io/gh/ahmedkamals/eventually/branch/master/graph/badge.svg)](https://codecov.io/gh/ahmedkamals/eventually "Code Coverage")
[![GolangCI](https://golangci.com/badges/github.com/ahmedkamals/eventually.svg?style=flat-square)](https://golangci.com/r/github.com/ahmedkamals/eventually "Code Coverage")
[![Go Report Card](https://goreportcard.com/badge/github.com/ahmedkamals/eventually)](https://goreportcard.com/report/github.com/ahmedkamals/eventually  "Go Report Card")
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/c282df1ff33c43ddb5da1d7fe4e85674)](https://www.codacy.com/app/ahmedkamals/eventually?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=ahmedkamals/eventually&amp;utm_campaign=Badge_Grade "Code Quality")
[![GoDoc](https://godoc.org/github.com/ahmedkamals/eventually?status.svg)](https://godoc.org/github.com/ahmedkamals/eventually "Documentation")
[![DepShield Badge](https://depshield.sonatype.org/badges/ahmedkamals/eventually/depshield.svg)](https://depshield.github.io "DepShield")
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fahmedkamals%2Feventually.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fahmedkamals%2Feventually?ref=badge_shield "Dependencies")
[![Join the chat at https://gitter.im/ahmedkamals/eventually](https://badges.gitter.im/ahmedkamals/eventually.svg)](https://gitter.im/ahmedkamals/eventually?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge "Let's discuss")

```bash
 _____                 _               _ _
| ____|_   _____ _ __ | |_ _   _  __ _| | |_   _
|  _| \ \ / / _ \ '_ \| __| | | |/ _` | | | | | |
| |___ \ V /  __/ | | | |_| |_| | (_| | | | |_| |
|_____| \_/ \___|_| |_|\__|\__,_|\__,_|_|_|\__, |
                                           |___/
```

is a library that helps to implement CQRS by having an event bus that facilitates the Pub-Sub operations.

Table of Contents
-----------------

*   [✨ Features](#-features)

*   [🏎️ Getting Started](#-getting-started)

    *   [Prerequisites](#prerequisites)
    *   [Installation](#installation)
    *   [Examples](#examples)

*   [🕸️ Tests](#-tests)

    *   [Benchmarks](#benchmarks)

*   [🤝 Contribution](#-contribution)

    *   [Git Hooks](#git-hooks)

*   [👨‍💻 Credits](#-credits)

*   [🆓 License](#-license)

✨ Features
-----------

*   Pub/Sub model.

*   Delayed publish.

*   Scheduled publish.

🏎️ Getting Started
------------------

### Prerequisites

*   [Golang 1.15 or later][1].

### Installation

```bash
go get -u github.com/ahmedkamals/eventually
```

### Examples

##### Main

```go
// main.go
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
```

##### Consumer

```go
// consumer.go
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
```

![Sample output](https://github.com/ahmedkamals/eventually/raw/master/assets/img/sample.gif "Sample output")

🕸️ Tests
--------

```bash
make test
```

### Benchmarks

![Benchmarks](https://github.com/ahmedkamals/eventually/raw/master/assets/img/benchmark.png "Benchmarks")
![Flamegraph](https://github.com/ahmedkamals/eventually/raw/master/assets/img/flamegraph.png "Benchmarks Flamegraph")

<details>
<summary>🔥 Todo:</summary>
   <ul>
       <li>Stats collection about uptime, and published messages.</li>
       <li>Storing published messages on a persistence medium (memory, disk) for a defined period of time.</li>
   </ul>
</details>

🤝 Contribution
---------------

Please refer to the [`CONTRIBUTING.md`](https://github.com/ahmedkamals/eventually/blob/master/CONTRIBUTING.md) file.

### Git Hooks

In order to set up tests running on each commit do the following steps:

```bash
ln -sf ../../assets/git/hooks/pre-commit.sh .git/hooks/pre-commit && \
ln -sf ../../assets/git/hooks/pre-push.sh .git/hooks/pre-push     && \
ln -sf ../../assets/git/hooks/commit-msg.sh .git/hooks/commit-msg
```

👨‍💻 Credits
----------

 * [ahmedkamals][2]

🆓 LICENSE
----------

Eventually is released under MIT license, please refer to the [`LICENSE.md`](https://github.com/ahmedkamals/eventually/blob/master/LICENSE.md) file.

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fahmedkamals%2Feventually.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fahmedkamals%2Feventually?ref=badge_large)

Happy Coding 🙂

[![Analytics](http://www.google-analytics.com/__utm.gif?utmwv=4&utmn=869876874&utmac=UA-136526477-1&utmcs=ISO-8859-1&utmhn=github.com&utmdt=Eventually&utmcn=1&utmr=0&utmp=/ahmedkamals/eventually?utm_source=www.github.com&utm_campaign=Eventually&utm_term=Eventually&utm_content=Eventually&utm_medium=repository&utmac=UA-136526477-1)]()

[1]: https://golang.org/dl/ "Download Golang"
[2]: https://github.com/ahmedkamals "Author"
