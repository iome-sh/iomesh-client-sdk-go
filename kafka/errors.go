package kafka

import "errors"

// ErrUnknownTopic is returned when a Kafka topic has no mapping.
var ErrUnknownTopic = errors.New("kafka: unknown topic")
