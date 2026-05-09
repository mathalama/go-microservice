package middleware

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type RateLimiter struct {
	client *redis.Client
	limit  int
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	limitStr := os.Getenv("RATE_LIMIT_RPM")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100 // Default
	}

	return &RateLimiter{
		client: client,
		limit:  limit,
	}
}

func (rl *RateLimiter) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		clientIP := rl.getClientIP(ctx)
		if clientIP == "" {
			return handler(ctx, req) // Allow if IP cannot be determined
		}

		key := fmt.Sprintf("ratelimit:%s", clientIP)
		now := time.Now().UnixNano()
		window := time.Minute

		// Use Redis pipeline to minimize round-trips
		pipe := rl.client.Pipeline()
		pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(now-window.Nanoseconds(), 10))
		pipe.ZCard(ctx, key)
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
		pipe.Expire(ctx, key, window)

		cmds, err := pipe.Exec(ctx)
		if err != nil && err != redis.Nil {
			log.Printf("Rate limiter error: %v", err)
			return handler(ctx, req) // Best effort
		}

		count := cmds[1].(*redis.IntCmd).Val()
		if int(count) > rl.limit {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded: %d rpm allowed. Retry after 60 seconds.", rl.limit)
		}

		return handler(ctx, req)
	}
}

func (rl *RateLimiter) getClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}
