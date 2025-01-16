package store

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
)

//go:generate mockgen -destination=mocks/mock_messagebroker.go -package=mockstore github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store PostMessageBrokerStore

// store interface for post broker events. In this case, to kafka
type PostMessageBrokerStore interface {
	Created(ctx context.Context, post model.Post) error
	Deleted(ctx context.Context, id int64) error
	Updated(ctx context.Context, post model.Post) error
}
