package suave

type Config struct {
	SuaveEthRemoteBackendEndpoint string
	RedisStorePubsubUri           string
	RedisStoreUri                 string
	PebbleDbPath                  string
	EthBundleSigningKeyHex        string
	EthBlockSigningKeyHex         string
	ExternalWhitelist             []string
}

var DefaultConfig = Config{}
