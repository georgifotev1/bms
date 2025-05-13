package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-redis/redis/v8"
)

type CustomerStore struct {
	rdb *redis.Client
}

func (s *CustomerStore) Get(ctx context.Context, customerID int64) (*store.Customer, error) {
	cacheKey := fmt.Sprintf("customer-%d", customerID)

	data, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var customer store.Customer
	if data != "" {
		err := json.Unmarshal([]byte(data), &customer)
		if err != nil {
			return nil, err
		}
	}

	return &customer, nil
}

func (s *CustomerStore) Set(ctx context.Context, customer *store.Customer) error {
	cacheKey := fmt.Sprintf("customer-%d", customer.ID)

	json, err := json.Marshal(customer)
	if err != nil {
		return err
	}

	return s.rdb.SetEX(ctx, cacheKey, json, UserExpTime).Err()
}

func (s *CustomerStore) Delete(ctx context.Context, customerID int64) {
	cacheKey := fmt.Sprintf("customer-%d", customerID)
	s.rdb.Del(ctx, cacheKey)
}
