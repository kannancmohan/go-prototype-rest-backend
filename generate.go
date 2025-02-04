// generate mocks for service in internal/api/service/mocks/ folder
//go:generate mockgen -destination=internal/api/service/mocks/mock_user_service.go -package=mockservice github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler UserService
//go:generate mockgen -destination=internal/api/service/mocks/mock_posts_service.go -package=mockservice github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler PostService

// generate mocks for store in internal/common/domain/store/mocks/ folder
//go:generate mockgen -destination=internal/common/domain/store/mocks/mock_db.go -package=mockstore github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store PostDBStore,RoleDBStore,UserDBStore
//go:generate mockgen -destination=internal/common/domain/store/mocks/mock_messagebroker.go -package=mockstore github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store PostMessageBrokerStore
//go:generate mockgen -destination=internal/common/domain/store/mocks/mock_search.go -package=mockstore github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store PostSearchStore,PostSearchIndexStore

package main

// This file is used to centralize go:generate directives.
// It does not need to contain any code other than the directives.