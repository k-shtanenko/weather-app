package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-shtanenko/weather-app/weather-api/config"
)

type BucketInfo struct {
	ShardIndex  int
	BucketIndex int
	ShardName   string
	Pool        *pgxpool.Pool
}

type BucketManager struct {
	shards       []*pgxpool.Pool
	buckets      []*BucketInfo
	totalBuckets int
	mu           sync.RWMutex
}

func NewBucketManager(ctx context.Context, cfg config.PostgresConfig) (*BucketManager, error) {
	shardURLs := make([]string, 0, len(cfg.ShardHosts))
	for _, host := range cfg.ShardHosts {
		connURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.User, cfg.Password, host, cfg.Port, cfg.Database, cfg.SSLMode)
		shardURLs = append(shardURLs, connURL)
	}

	if len(shardURLs) == 0 {
		return nil, fmt.Errorf("no shards configured")
	}

	shardPools := make([]*pgxpool.Pool, len(shardURLs))
	for i, connURL := range shardURLs {
		pool, err := newShardPool(ctx, connURL, cfg)
		if err != nil {
			for j := 0; j < i; j++ {
				shardPools[j].Close()
			}
			return nil, fmt.Errorf("create shard %d: %w", i, err)
		}
		shardPools[i] = pool
	}

	var buckets []*BucketInfo
	bucketIndex := 0

	bucketsInShard := 1
	for shardIndex := range shardURLs {
		for bucketIdx := 0; bucketIdx < bucketsInShard; bucketIdx++ {
			buckets = append(buckets, &BucketInfo{
				ShardIndex:  shardIndex,
				BucketIndex: bucketIdx,
				ShardName:   fmt.Sprintf("shard_%d", shardIndex),
				Pool:        shardPools[shardIndex],
			})
			bucketIndex++
		}
	}

	return &BucketManager{
		shards:       shardPools,
		buckets:      buckets,
		totalBuckets: len(buckets),
	}, nil
}

func newShardPool(ctx context.Context, connURL string, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	config.MaxConns = int32(cfg.MaxConnections)
	config.MinConns = int32(cfg.MaxIdleConnections)
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.ConnConfig.ConnectTimeout = cfg.ConnectionTimeout

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	return pool, nil
}

func (bm *BucketManager) GetBucketForUser(userID int) *BucketInfo {
	bucketIndex := bm.GetBucketIndex(userID)
	return bm.buckets[bucketIndex]
}

func (bm *BucketManager) GetBucketIndex(userID int) int {
	id := userID
	if id < 0 {
		id = -id
	}
	return id % bm.totalBuckets
}

func (bm *BucketManager) GetPoolForUser(userID int) *pgxpool.Pool {
	bucket := bm.GetBucketForUser(userID)
	return bucket.Pool
}

func (bm *BucketManager) GetBucketSchema(userID int) string {
	bucket := bm.GetBucketForUser(userID)
	return fmt.Sprintf("bucket_%d_%d", bucket.ShardIndex, bucket.BucketIndex)
}

func (bm *BucketManager) Close() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, pool := range bm.shards {
		if pool != nil {
			pool.Close()
		}
	}
}
