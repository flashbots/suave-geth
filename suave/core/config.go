package suave

import "time"

type Config struct {
	SuaveEthRemoteBackendEndpoint string // deprecated
	RedisStorePubsubUri           string
	RedisStoreUri                 string
	RedisStoreTTL                 time.Duration
	PebbleDbPath                  string
	EthBundleSigningKeyHex        string
	EthBlockSigningKeyHex         string
	ExternalWhitelist             []string
	AliasRegistry                 map[string]string
}

var DefaultConfig = Config{}
