package suave

type Config struct {
	SuaveEthRemoteBackendEndpoint string
	RedisStorePubsubUri           string
	RedisStoreUri                 string
}

var DefaultConfig = Config{}
