package eventually

import (
	"bytes"
	"encoding/gob"
	"github.com/ahmedkamals/eventually/internal/errors"
)

type (
	// Serializer interface
	Serializer interface {
		serialize(interface{}) ([]byte, error)
		deserialize([]byte, interface{}) error
	}
	serializer struct{}
)

// NewSerializer creates a new Serializer
func NewSerializer() Serializer {
	return &serializer{}
}

func (s *serializer) serialize(target interface{}) ([]byte, error) {
	const op errors.Operation = "serializer.serialize"

	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	if err := encoder.Encode(target); err != nil {
		return nil, errors.E(op, errors.Failure, err)
	}

	return buffer.Bytes(), nil
}

func (s *serializer) deserialize(data []byte, target interface{}) error {
	buffer := bytes.Buffer{}
	buffer.Write(data)
	decoder := gob.NewDecoder(&buffer)

	return decoder.Decode(target)
}
