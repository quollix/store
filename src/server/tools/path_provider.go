package tools

import (
	"path/filepath"

	u "github.com/quollix/common/utils"
)

type PathProviderImpl struct {
	assetsDir string
}

func (p *PathProviderImpl) Initialize() error {
	var err error
	p.assetsDir, err = u.FindDir("assets")
	return err
}

func (p *PathProviderImpl) GetAssetsDir() string {
	return p.assetsDir
}

func (p *PathProviderImpl) GetMigrationsDir() string {
	return filepath.Join(p.assetsDir, "/migrations")
}
