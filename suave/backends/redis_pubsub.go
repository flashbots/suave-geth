package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flashbots/go-utils/cli"
	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

var (
	redisUpsertTopic = "store:upsert"

	redisConnectionPoolSize = cli.GetEnvInt("REDIS_CONNECTION_POOL_SIZE", 0) // 0 means use default (10 per CPU)
	redisMinIdleConnections = cli.GetEnvInt("REDIS_MIN_IDLE_CONNECTIONS", 0) // 0 means use default
	redisReadTimeoutSec     = cli.GetEnvInt("REDIS_READ_TIMEOUT_SEC", 0)     // 0 means use default (3 sec)
	redisPoolTimeoutSec     = cli.GetEnvInt("REDIS_POOL_TIMEOUT_SEC", 0)     // 0 means use default (ReadTimeout + 1 sec)
	redisWriteTimeoutSec    = cli.GetEnvInt("REDIS_WRITE_TIMEOUT_SEC", 0)    // 0 means use default (3 seconds)

	formatRedisBidKey = func(bidId suave.BidId) string {
		return fmt.Sprintf("bid-%x", bidId)
	}

	formatRedisBidValueKey = func(bid suave.Bid, key string) string {
		return fmt.Sprintf("bid-data-%x-%s", bid.Id, key) // TODO: should also include the hash of the bid at least
	}
)

type RedisPubSub struct {
	ctx    context.Context
	client *redis.Client
	pubsub *redis.PubSub
}

func NewRedisPubSub(ctx context.Context, redisURI string) (*RedisPubSub, error) {
	client, err := connectRedis(redisURI)
	if err != nil {
		return nil, err
	}

	pubsub := client.Subscribe(ctx, redisUpsertTopic)

	go func() {
		<-ctx.Done()
		pubsub.Close()
		client.Close()
	}()

	return &RedisPubSub{
		ctx:    ctx,
		client: client,
		pubsub: pubsub,
	}, nil
}

func (r *RedisPubSub) Subscribe() <-chan suave.DAMessage {
	ch := make(chan suave.DAMessage, 16)

	go func() {
		for r.ctx.Err() == nil {
			rmsg, err := r.pubsub.ReceiveMessage(r.ctx)
			if err != nil {
				continue
			}

			var msg suave.DAMessage
			err = json.Unmarshal([]byte(rmsg.Payload), &msg)
			if err != nil {
				log.Debug("could not parse message from subscription", "err", err, "msg", rmsg.Payload)
			}

			// For some reason the caller, key, and value fields are not parsed correctly
			// TODO: debug
			m := make(map[string]interface{})
			err = json.Unmarshal([]byte(rmsg.Payload), &m)
			if err != nil {
				log.Debug("could not parse message from subscription", "err", err, "msg", rmsg.Payload)
			}

			msg.Caller = common.HexToAddress(m["caller"].(string))
			msg.Key = m["key"].(string)
			msg.Value = common.FromHex(m["value"].(string))

			select {
			case ch <- msg:
				continue
			default:
				log.Warn("dropping transport message due to channel being blocked")
				continue
			}
		}
	}()

	return ch
}

func (r *RedisPubSub) Publish(message suave.DAMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		panic(fmt.Errorf("could not marshal message: %w", err))
	}

	r.client.Publish(r.ctx, miniredisUpsertTopic, string(data))
}

func connectRedis(redisURI string) (*redis.Client, error) {
	// Handle both URIs and full URLs, assume unencrypted connections
	if !strings.HasPrefix(redisURI, "redis://") && !strings.HasPrefix(redisURI, "rediss://") {
		redisURI = "redis://" + redisURI
	}

	redisOpts, err := redis.ParseURL(redisURI)
	if err != nil {
		return nil, err
	}

	if redisConnectionPoolSize > 0 {
		redisOpts.PoolSize = redisConnectionPoolSize
	}
	if redisMinIdleConnections > 0 {
		redisOpts.MinIdleConns = redisMinIdleConnections
	}
	if redisReadTimeoutSec > 0 {
		redisOpts.ReadTimeout = time.Duration(redisReadTimeoutSec) * time.Second
	}
	if redisPoolTimeoutSec > 0 {
		redisOpts.PoolTimeout = time.Duration(redisPoolTimeoutSec) * time.Second
	}
	if redisWriteTimeoutSec > 0 {
		redisOpts.WriteTimeout = time.Duration(redisWriteTimeoutSec) * time.Second
	}

	redisClient := redis.NewClient(redisOpts)
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		// unable to connect to redis
		return nil, err
	}
	return redisClient, nil
}
