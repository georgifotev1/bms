package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/georgifotev1/bms/internal/store"
	"github.com/go-redis/redis/v8"
)

type BrandStore struct {
	rdb *redis.Client
}

func (s *BrandStore) Get(ctx context.Context, brandID int32) (*store.BrandResponse, error) {
	cacheKey := fmt.Sprintf("brand-%d", brandID)

	data, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var brand store.BrandResponse
	if data != "" {
		err := json.Unmarshal([]byte(data), &brand)
		if err != nil {
			return nil, err
		}
	}

	return &brand, nil
}

func (s *BrandStore) Set(ctx context.Context, brand *store.BrandResponse) error {
	cacheKey := fmt.Sprintf("brand-%d", brand.ID)

	json, err := json.Marshal(brand)
	if err != nil {
		return err
	}

	return s.rdb.SetEX(ctx, cacheKey, json, UserExpTime).Err()
}

func (s *BrandStore) Delete(ctx context.Context, brandId int32) {
	cacheKey := fmt.Sprintf("brand-%d", brandId)
	s.rdb.Del(ctx, cacheKey)
}
