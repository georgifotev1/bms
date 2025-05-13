package cache

import (
	"context"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-redis/redis/v8"
)

type Storage struct {
	Users interface {
		Get(context.Context, int64) (*store.User, error)
		Set(context.Context, *store.User) error
		Delete(context.Context, int64)
	}
	Brands interface {
		Get(context.Context, int32) (*store.BrandResponse, error)
		Set(context.Context, *store.BrandResponse) error
		Delete(context.Context, int32)
	}
	Customers interface {
		Get(context.Context, int64) (*store.Customer, error)
		Set(context.Context, *store.Customer) error
		Delete(context.Context, int64)
	}
}

func NewRedisStorage(rbd *redis.Client) Storage {
	return Storage{
		Users:     &UserStore{rdb: rbd},
		Brands:    &BrandStore{rdb: rbd},
		Customers: &CustomerStore{rdb: rbd},
	}
}
