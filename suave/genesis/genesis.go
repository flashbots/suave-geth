package genesis

import (
	"embed"
	"io/fs"
	"os"
	"strings"
)

//go:embed fixtures
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

	// load 'name' from the embedded files
	fsys, err := fs.Sub(genesisFiles, "fixtures")
	if err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(fsys, name+".json")
	if err != nil {
		return nil, err
	}
	return data, nil
}
