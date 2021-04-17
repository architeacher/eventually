package eventually

import (
	"context"
	"testing"
	"time"
)

func BenchmarkPublish(b *testing.B) {
	topics := [...]Descriptor{
		newTestDescriptor(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		eventBus := newEventBus(ctx, 1000)

		for pb.Next() {
			for _, topic := range topics {
				eventBus.Publish(topic)
			}
		}
	})
}

func BenchmarkPublishAfter(b *testing.B) {
	topics := [...]Descriptor{
		newTestDescriptor(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		eventBus := newEventBus(ctx, 1000)

		for pb.Next() {
			for _, topic := range topics {
				eventBus.PublishAfter(ctx, time.Nanosecond, topic)
			}
		}
	})
}

func BenchmarkSchedulePublish(b *testing.B) {
	topics := []Descriptor{
		newTestDescriptor(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ticker := time.NewTicker(time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		eventBus := newEventBus(ctx, 1000)

		for pb.Next() {
			eventBus.SchedulePublish(ctx, ticker, topics...)
		}
	})
}

func BenchmarkSubscribe(b *testing.B) {
	topics := []Descriptor{
		newTestDescriptor(),
		newTestDescriptor(),
	}
	consumers := getFakeConsumers(1)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		eventBus := newEventBus(ctx, 1000)

		for pb.Next() {
			for _, consumer := range consumers {
				err := eventBus.Subscribe(consumer, topics...)
				if err != nil {
					b.Log(err)
				}
			}
		}
	})
}

func BenchmarkUnsubscribe(b *testing.B) {
	topics := []Descriptor{
		newTestDescriptor(),
		newTestDescriptor(),
	}
	consumers := getFakeConsumers(10)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		eventBus := newEventBus(ctx, 1000)

		for pb.Next() {
			for _, consumer := range consumers {
				err := eventBus.Unsubscribe(consumer, topics...)
				if err != nil {
					b.Log(err)
				}
			}
		}
	})
}

func BenchmarkUnregister(b *testing.B) {
	topics := []Descriptor{
		newTestDescriptor(),
		newTestDescriptor(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventBus := newEventBus(ctx, 1000)

		for pb.Next() {
			err := eventBus.Unregister(topics...)
			if err != nil {
				b.Log(err)
			}
		}
	})
}
