package cstore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/flashbots/go-utils/cli"
	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	redisUpsertTopic = "store:upsert"

	redisConnectionPoolSize = cli.GetEnvInt("REDIS_CONNECTION_POOL_SIZE", 0) // 0 means use default (10 per CPU)
	redisMinIdleConnections = cli.GetEnvInt("REDIS_MIN_IDLE_CONNECTIONS", 0) // 0 means use default
	redisReadTimeoutSec     = cli.GetEnvInt("REDIS_READ_TIMEOUT_SEC", 0)     // 0 means use default (3 sec)
	redisPoolTimeoutSec     = cli.GetEnvInt("REDIS_POOL_TIMEOUT_SEC", 0)     // 0 means use default (ReadTimeout + 1 sec)
	redisWriteTimeoutSec    = cli.GetEnvInt("REDIS_WRITE_TIMEOUT_SEC", 0)    // 0 means use default (3 seconds)
)

type RedisPubSubTransport struct {
	ctx      context.Context
	cancel   context.CancelFunc
	redisUri string
	client   *redis.Client
}

func NewRedisPubSubTransport(redisUri string) *RedisPubSubTransport {
	return &RedisPubSubTransport{
		redisUri: redisUri,
	}
}

func (r *RedisPubSubTransport) Start() error {
	if r.cancel != nil {
		r.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	r.cancel = cancel

	client, err := connectRedis(r.redisUri)
	if err != nil {
		return err
	}
	r.client = client

	return nil
}

func (r *RedisPubSubTransport) Stop() error {
	if r.cancel == nil || r.client == nil {
		return errors.New("Redis pubsub: Stop() called before Start()")
	}

	r.cancel()
	r.client.Close()

	return nil
}

func (r *RedisPubSubTransport) Subscribe() (<-chan DAMessage, context.CancelFunc) {
	ch := make(chan DAMessage, 16)
	ctx, cancel := context.WithCancel(r.ctx)

	// Each subscriber has its own PubSub as it blocks on receive!
	pubsub := r.client.Subscribe(ctx, redisUpsertTopic)

	go func() {
		defer close(ch)
		defer pubsub.Close()

		for ctx.Err() == nil /* run until Stop() or cancel() called */ {
			rmsg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// Stop() or cancel() called, simply exit
					log.Info("Redis pubsub: closing subscription")
					return
				}

				// Reconnection is not necessary as it's handled by redis.PubSub, simply log the error and continue
				log.Error("Redis pubsub: error while receiving messages", "err", err)
				continue
			}

			var msg DAMessage
			msgBytes := common.Hex2Bytes(rmsg.Payload)
			if err != nil {
				log.Trace("Redis pubsub: could not decode message from subscription", "err", err, "msg", rmsg.Payload)
				continue
			}

			err = json.Unmarshal(msgBytes, &msg)
			if err != nil {
				log.Trace("Redis pubsub: could not parse message from subscription", "err", err, "msg", rmsg.Payload)
				continue
			}

			// For some reason the caller, key, and value fields are not parsed correctly
			// TODO: debug
			m := make(map[string]interface{})
			err = json.Unmarshal(msgBytes, &m)
			if err != nil {
				log.Trace("Redis pubsub: could not parse message from subscription", "err", err, "msg", rmsg.Payload)
				continue
			}

			/*
				msg.Caller = common.HexToAddress(m["caller"].(string))
				msg.Key = m["key"].(string)
				msg.Value = common.FromHex(m["value"].(string))
			*/

			log.Debug("Redis pubsub: new message", "msg", msg)
			select {
			case <-ctx.Done():
				log.Info("Redis pubsub: closing subscription")
				return
			case ch <- msg:
				continue
			default:
				log.Error("dropping transport message due to channel being blocked")
				continue
			}
		}
	}()

	return ch, cancel
}

func (r *RedisPubSubTransport) Publish(message DAMessage) {
	log.Trace("Redis pubsub: publishing", "message", message)
	data, err := json.Marshal(message)
	if err != nil {
		log.Error("Redis pubsub: could not marshal message", "err", err)
		return
	}

	r.client.Publish(r.ctx, redisUpsertTopic, common.Bytes2Hex(data))
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
