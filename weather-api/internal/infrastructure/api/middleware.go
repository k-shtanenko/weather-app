package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
	"golang.org/x/time/rate"
)

type Middleware struct {
	logger      logger.Logger
	rateLimiter *rate.Limiter
}

func NewMiddleware(rateLimit int, rateWindow time.Duration) *Middleware {
	return &Middleware{
		logger:      logger.New("info", "development").WithField("component", "middleware"),
		rateLimiter: rate.NewLimiter(rate.Every(rateWindow), rateLimit),
	}
}

func (m *Middleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (m *Middleware) Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		_ = c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				m.logger.Error(e)
			}
		} else {
			m.logger.Infof("HTTP | %3d | %13v | %15s | %-7s %s",
				c.Writer.Status(),
				latency,
				c.ClientIP(),
				c.Request.Method,
				path,
			)
		}
	}
}

func (m *Middleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.rateLimiter.Allow() {
			m.logger.Warnf("Rate limit exceeded for IP: %s", c.ClientIP())
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *Middleware) Cache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "public, max-age=3600")
		c.Next()
	}
}

func (m *Middleware) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Errorf("Panic recovered: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "An unexpected error occurred",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
