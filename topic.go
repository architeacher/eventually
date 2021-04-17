package eventually

import (
	"encoding/json"
	"fmt"
	"github.com/ahmedkamals/eventually/internal/errors"
	"github.com/google/uuid"
	"sync"
	"time"
)

type (
	// UUID of the topic.
	UUID string

	// Category of the topic.
	Category string

	// Version of the topic.
	Version int

	// Metadata about the topic.
	Metadata map[string]interface{}

	// Payload of the topic.
	Payload interface{}

	identifiable interface {
		// AggregateID returns the id of topic's related Aggregate.
		AggregateID() UUID
	}

	versional interface {
		// Version returns the version of the topic.
		Version() Version
	}

	timeAware interface {
		// CreatedAt returns the time when the topic was created.
		CreatedAt() time.Time
	}

	decorative interface {
		// GetHeaders returns the MetaData for the topic.
		GetHeaders() Metadata
		// SetHeader sets the value of the metadata specified by the key.
		SetHeader(string, interface{})
		// GetHeader returns the value of the metadata specified by the key.
		GetHeader(string) (interface{}, bool)
	}

	carrier interface {
		// Payload return the actual payload of the topic message.
		Payload() Payload
	}

	marshaller interface {
		// Marshal returns the JSON encoding of a topic.
		Marshal() (string, error)
	}

	unmarshaller interface {
		// Unmarshal parses the JSON-encoded topic.
		Unmarshal(data string) error
	}

	serializable interface {
		// Serialize converts the current topic to a serialized format.
		Serialize() (string, error)
	}

	unserializable interface {
		// Deserialize converts serialized format to a topic.
		Deserialize(data string) error
	}

	// Descriptor is the interface that a topic must implement.
	Descriptor interface {
		identifiable
		versional
		timeAware
		decorative
		carrier
		marshaller
		unmarshaller
		serializable
		unserializable
		fmt.GoStringer
		fmt.Stringer

		// Name returns the topic category.
		Name() Category
	}

	// descriptor is an implementation of the topic Descriptor interface.
	descriptor struct {
		mtx            sync.RWMutex
		ID             UUID      `json:"id,omitempty"`
		TopicName      Category  `json:"name,omitempty"`
		EventVersion   Version   `json:"version,omitempty"`
		EventCreatedAt time.Time `json:"created_at,omitempty"`
		MetaData       Metadata  `json:"metadata"`
		EventPayload   Payload   `json:"payload,omitempty"`
		serializer     Serializer
	}
)

const (
	// universalTopicUUID used for subscribing to all the topicsMap.
	universalTopicUUID UUID = "*"
)

// NewUUID creates new UUID.
func NewUUID() UUID {
	return UUID(uuid.New().String())
}

func newDescriptor(aggregateID UUID, category Category, payload Payload, version Version) Descriptor {
	return &descriptor{
		ID:             aggregateID,
		TopicName:      category,
		EventPayload:   payload,
		EventVersion:   version,
		EventCreatedAt: time.Now(),
		MetaData:       make(Metadata),
		serializer:     NewSerializer(),
	}
}

// NewDescriptor returns a new topic descriptor.
func NewDescriptor(category Category, payload Payload, version Version) Descriptor {
	return newDescriptor(NewUUID(), category, payload, version)
}

// newUniversalDescriptor returns a new universal topic descriptor.
func newUniversalDescriptor(category Category, payload Payload, version Version) Descriptor {
	return newDescriptor(universalTopicUUID, category, payload, version)
}

func (d *descriptor) AggregateID() UUID {
	return d.ID
}

func (d *descriptor) Name() Category {
	return d.TopicName
}

func (d *descriptor) Version() Version {
	return d.EventVersion
}

func (d *descriptor) GoString() string {
	return fmt.Sprintf("%s[%s]", d.String(), d.ID)
}

func (d *descriptor) String() string {
	return fmt.Sprintf("%s@v%d", d.Name(), d.Version())
}

func (d *descriptor) CreatedAt() time.Time {
	return d.EventCreatedAt
}

func (d *descriptor) GetHeaders() Metadata {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	return d.MetaData
}

func (d *descriptor) SetHeader(key string, value interface{}) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.MetaData[key] = value
}

func (d *descriptor) GetHeader(key string) (interface{}, bool) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	item, ok := d.MetaData[key]

	return item, ok
}

func (d *descriptor) Payload() Payload {
	return d.EventPayload
}

func (d *descriptor) Marshal() (string, error) {
	const op errors.Operation = "Topic.Marshal"
	encodedData, err := json.Marshal(d)

	if err != nil {
		return "", errors.E(op, errors.Failure, err)
	}

	return string(encodedData), nil
}

func (d *descriptor) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), d)
}

func (d *descriptor) Serialize() (string, error) {
	const op errors.Operation = "Topic.Serialize"
	serializedData, err := d.serializer.serialize(d)

	if err != nil {
		return "", errors.E(op, errors.Failure, err)
	}

	return string(serializedData), nil
}

func (d *descriptor) Deserialize(data string) error {
	return d.serializer.deserialize([]byte(data), d)
}
