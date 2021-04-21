package eventually

type (
	// Inbox interface for consumers to receive messages on.
	Inbox interface {
		Receive(Descriptor)
		Read() Descriptor
		Signout()
	}

	inbox struct {
		messagesChan chan Descriptor
	}
)

func (i *inbox) Receive(topic Descriptor) {
	select {
	case i.messagesChan <- topic:
	//Drop any message that exceeds the mail box space.
	default:
	}
}

func (i *inbox) Read() Descriptor {
	return <-i.messagesChan
}

func (i *inbox) Signout() {
	close(i.messagesChan)
}

// NewInbox creates a new Inbox.
func NewInbox(space uint) Inbox {
	return &inbox{
		messagesChan: make(chan Descriptor, space),
	}
}
