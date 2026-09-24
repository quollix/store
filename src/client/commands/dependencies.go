package commands

import (
	"qsc/configuration"
	"qsc/local"
	"qsc/remote"
	"qsc/tools"

	"github.com/quollix/common/store"
	"github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

type ClientDependencies struct {
	Config                *tools.GlobalConfig
	ConfigProvider        configuration.Provider
	SigningKeyManager     remote.SigningKeyManager
	SessionManager        remote.SessionManager
	AppStoreClient        *store.AppStoreClientImpl
	VersionSigningService store.VersionSigningService
	VersionValidator      validation.VersionValidator
	OsWrapper             utils.OsWrapper
	Updater               *local.Updater
	ConsistencyChecker    local.ConsistencyChecker
	AppSelector           local.AppSelector
	FileSystemOperator    local.FileSystemOperator
}
