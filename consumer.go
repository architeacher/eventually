package eventually

import "fmt"

type (
	// Match callback that can be used for matching event.
	Match func(Descriptor) bool
	// Notification callback when a subscription on topic happens (e.g. OnSubscribe, On.Unsubscribe).
	Notification func(Descriptor, Consumer)

	// Consumer interface defines consumers operations.
	Consumer interface {
		identifiable
		fmt.GoStringer
		fmt.Stringer
		// MatchCriteria returns a match criteria callback.
		MatchCriteria() Match
		// Drop puts the message in the receiver mail box.
		Drop(Descriptor)
		// ReadMessage reads a message from the receiver mail box.
		ReadMessage() Descriptor
		// OnSubscribe called on new subscription.
		OnSubscribe(Descriptor, Notification)
		// OnUnsubscribe called on subscription cancellation.
		OnUnsubscribe(Descriptor, Notification)
		// Signout closes the mailbox.
		Signout()
	}
)
