package suave

type Config struct {
	SuaveEthRemoteBackendEndpoint string
	RedisStorePubsubUri           string
	RedisStoreUri                 string
	EthBundleSigningKeyHex        string
	EthBlockSigningKeyHex         string
}

var DefaultConfig = Config{}
