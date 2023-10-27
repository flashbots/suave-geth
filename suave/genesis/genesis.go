package genesis

import (
	"embed"
	"os"
	"strings"
)

// Right now there is only a test.json file because
// the binary does not compile if the embed.FS is empty.
//
//go:embed *.json
var genesisFiles embed.FS

func Load(name string) ([]byte, error) {
	if strings.HasSuffix(name, ".json") {
		// load the genesis file from the file system
		genesisRaw, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		return genesisRaw, nil
	}

	// load 'name' frrom the embedded files
	data, err := genesisFiles.ReadFile(name + ".json")
	if err != nil {
		return nil, err
	}
	return data, nil
}
