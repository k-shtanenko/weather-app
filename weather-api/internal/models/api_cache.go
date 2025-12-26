package models

import (
	"time"
)

type APICacheEntity interface {
	GetID() string
	GetCacheKey() string
	GetCacheType() string
	GetData() []byte
	GetContentType() string
	GetFileName() string
	GetExpiresAt() time.Time
	GetHitCount() int
	GetCreatedAt() time.Time
	GetLastAccessedAt() time.Time
	IsExpired() bool
	IncrementHitCount()
	UpdateLastAccessed()
}

type APICache struct {
	ID             string    `json:"id" db:"id"`
	CacheKey       string    `json:"cache_key" db:"cache_key"`
	CacheType      string    `json:"cache_type" db:"cache_type"`
	Data           []byte    `json:"data" db:"data"`
	ContentType    string    `json:"content_type" db:"content_type"`
	FileName       string    `json:"file_name,omitempty" db:"file_name"`
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	HitCount       int       `json:"hit_count" db:"hit_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at" db:"last_accessed_at"`
}

func (c *APICache) GetID() string                { return c.ID }
func (c *APICache) GetCacheKey() string          { return c.CacheKey }
func (c *APICache) GetCacheType() string         { return c.CacheType }
func (c *APICache) GetData() []byte              { return c.Data }
func (c *APICache) GetContentType() string       { return c.ContentType }
func (c *APICache) GetFileName() string          { return c.FileName }
func (c *APICache) GetExpiresAt() time.Time      { return c.ExpiresAt }
func (c *APICache) GetHitCount() int             { return c.HitCount }
func (c *APICache) GetCreatedAt() time.Time      { return c.CreatedAt }
func (c *APICache) GetLastAccessedAt() time.Time { return c.LastAccessedAt }

func (c *APICache) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (c *APICache) IncrementHitCount() {
	c.HitCount++
}

func (c *APICache) UpdateLastAccessed() {
	c.LastAccessedAt = time.Now()
}

type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Reason
}
