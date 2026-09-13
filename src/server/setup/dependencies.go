package setup

import (
	"server/apps"
	"server/maintainers"
	"server/tools"
	"server/versions"

	u "github.com/quollix/common/utils"
)

type InitializerDependencies struct {
	HandlerInitializer         *HandlerInitializer
	DatabaseProvider           *tools.DatabaseProviderImpl
	DatabaseSnapshotRepository *u.DatabaseSnapshotRepositoryImpl
	PathProvider               *tools.PathProviderImpl
	Server                     *Server
	Config                     *tools.Config
	UserService                *maintainers.UserServiceImpl
	UserRepo                   *maintainers.UserRepositoryImpl
	EmailConfigRepo            *maintainers.EmailConfigRepositoryImpl
	AppRepo                    *apps.AppRepositoryImpl
	VersionRepo                *versions.VersionRepositoryImpl
}
