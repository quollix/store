package main

import (
	"os"
	"server/di"
	"server/setup"

	u "github.com/quollix/common/utils"
)

func main() {
	deps := di.WireDependencies()
	err := initializeApplication(deps)
	if err != nil {
		u.Logger.Error(err, "result", "failed to start application")
		os.Exit(1)
	}
}

func initializeApplication(deps *setup.InitializerDependencies) error {
	if err := deps.PathProvider.Initialize(); err != nil {
		return err
	}
	if err := deps.DatabaseProvider.InitializeDatabase(); err != nil {
		return err
	}
	if err := deps.UserService.InitializeAdminAccountIfNotExisting(); err != nil {
		return err
	}
	if deps.Config.SeedSampleData {
		if err := setup.SeedSampleData(deps); err != nil {
			return err
		}
	}
	if err := deps.DatabaseSnapshotRepository.DeleteDatabaseSnapshotIfExists(); err != nil {
		return err
	}
	if deps.Config.OpenWipeEndpoint {
		if err := deps.DatabaseSnapshotRepository.CreateDatabaseSnapshot(); err != nil {
			return err
		}
	}
	deps.HandlerInitializer.InitializeHandlers()
	return deps.Server.Run()
}
