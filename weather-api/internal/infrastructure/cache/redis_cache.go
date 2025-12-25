package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
)

type RedisCache struct {
	client *redis.Client
	logger logger.Logger
}

func NewRedisCache(host string, port int, password string, db int) (*RedisCache, error) {
	log := logger.New("info", "development").WithField("component", "redis_cache")

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Redis cache initialized successfully")
	return &RedisCache{
		client: client,
		logger: log,
	}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (entities.APICacheEntity, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	metaKey := r.getMetaKey(key)
	meta, err := r.client.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	expiresAt, _ := time.Parse(time.RFC3339, meta["expires_at"])
	createdAt, _ := time.Parse(time.RFC3339, meta["created_at"])
	lastAccessedAt, _ := time.Parse(time.RFC3339, meta["last_accessed_at"])

	cacheEntry := &entities.APICache{
		ID:             meta["id"],
		CacheKey:       key,
		CacheType:      meta["cache_type"],
		Data:           data,
		ContentType:    meta["content_type"],
		FileName:       meta["file_name"],
		ExpiresAt:      expiresAt,
		HitCount:       parseInt(meta["hit_count"]),
		CreatedAt:      createdAt,
		LastAccessedAt: lastAccessedAt,
	}

	go r.updateAccessStats(ctx, key)

	return cacheEntry, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, data entities.APICacheEntity, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, data.GetData(), ttl).Err(); err != nil {
		return fmt.Errorf("failed to set data in Redis: %w", err)
	}

	metaKey := r.getMetaKey(key)
	meta := map[string]interface{}{
		"id":               data.GetID(),
		"cache_type":       data.GetCacheType(),
		"content_type":     data.GetContentType(),
		"file_name":        data.GetFileName(),
		"expires_at":       data.GetExpiresAt().Format(time.RFC3339),
		"hit_count":        data.GetHitCount(),
		"created_at":       data.GetCreatedAt().Format(time.RFC3339),
		"last_accessed_at": data.GetLastAccessedAt().Format(time.RFC3339),
	}

	if err := r.client.HSet(ctx, metaKey, meta).Err(); err != nil {
		return fmt.Errorf("failed to set metadata in Redis: %w", err)
	}

	if err := r.client.Expire(ctx, metaKey, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set metadata TTL: %w", err)
	}

	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	metaKey := r.getMetaKey(key)
	if err := r.client.Del(ctx, metaKey).Err(); err != nil {
		return fmt.Errorf("failed to delete metadata from Redis: %w", err)
	}

	return nil
}

func (r *RedisCache) DeleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var deletedCount int64

	for {
		var keys []string
		var err error
		keys, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			count, err := r.client.Del(ctx, keys...).Result()
			if err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
			deletedCount += count

			for _, key := range keys {
				metaKey := r.getMetaKey(key)
				r.client.Del(ctx, metaKey)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

func (r *RedisCache) HealthCheck(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis ping failed: %w", err)
	}

	testKey := "healthcheck:" + time.Now().Format("20060102150405")
	testValue := "test"

	if err := r.client.Set(ctx, testKey, testValue, 10*time.Second).Err(); err != nil {
		return fmt.Errorf("Redis set test failed: %w", err)
	}

	val, err := r.client.Get(ctx, testKey).Result()
	if err != nil {
		return fmt.Errorf("Redis get test failed: %w", err)
	}

	if val != testValue {
		return fmt.Errorf("Redis test value mismatch")
	}

	return nil
}

func (r *RedisCache) Close() error {
	r.logger.Info("Closing Redis cache...")
	return r.client.Close()
}

func (r *RedisCache) getMetaKey(key string) string {
	return key + ":meta"
}

func (r *RedisCache) updateAccessStats(ctx context.Context, key string) {
	metaKey := r.getMetaKey(key)
	r.client.HIncrBy(ctx, metaKey, "hit_count", 1)
	r.client.HSet(ctx, metaKey, "last_accessed_at", time.Now().Format(time.RFC3339))
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
