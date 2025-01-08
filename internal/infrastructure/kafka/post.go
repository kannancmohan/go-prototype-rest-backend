package kafka

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
)

type event struct {
	Type  string
	Value model.Post
}

type postMessageBrokerStore struct {
	producer  *kafka.Producer
	topicName string
}

func NewPostMessageBrokerStore(producer *kafka.Producer, topicName string) *postMessageBrokerStore {
	return &postMessageBrokerStore{
		producer:  producer,
		topicName: topicName,
	}
}

func (p *postMessageBrokerStore) Created(ctx context.Context, post model.Post) error {
	return p.publish("posts.event.created", post)
}

func (p *postMessageBrokerStore) Deleted(ctx context.Context, id int64) error {
	return p.publish("posts.event.deleted", model.Post{ID: id})
}

func (p *postMessageBrokerStore) Updated(ctx context.Context, post model.Post) error {
	return p.publish("posts.event.updated", post)
}

func (p *postMessageBrokerStore) publish(msgType string, post model.Post) error {
	var b bytes.Buffer

	evt := event{
		Type:  msgType,
		Value: post,
	}

	if err := json.NewEncoder(&b).Encode(evt); err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "json.Encode")
	}

	if err := p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &p.topicName,
			Partition: kafka.PartitionAny,
		},
		Value: b.Bytes(),
	}, nil); err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "product.Producer")
	}

	return nil
}
