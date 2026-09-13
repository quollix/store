//go:build wireinject

package di

import (
	"net/http"
	"server/apps"
	"server/maintainers"
	"server/setup"
	"server/tools"
	"server/versions"

	"github.com/google/wire"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

func WireDependencies() *setup.InitializerDependencies {
	wire.Build(
		NewDatabaseProvider,
		NewDatabaseSnapshotRepository,
		tools.NewConfig,
		wire.Struct(new(tools.PathProviderImpl)),
		NewMux,
		NewEmailClient,
		NewRegistrationCodeProvider,
		NewVersionValidator,

		wire.Struct(new(maintainers.EmailServiceImpl), "*"),
		wire.Struct(new(maintainers.EmailHandler), "*"),
		wire.Struct(new(setup.InitializerDependencies), "*"),
		wire.Struct(new(apps.AppsHandlerImpl), "*"),
		wire.Struct(new(apps.AppRepositoryImpl), "*"),
		wire.Struct(new(setup.HandlerInitializer), "*"),
		wire.Struct(new(versions.VersionsHandler), "*"),
		wire.Struct(new(versions.VersionRepositoryImpl), "*"),
		wire.Struct(new(maintainers.UserRepositoryImpl), "*"),
		wire.Struct(new(maintainers.UserHandler), "*"),
		wire.Struct(new(maintainers.EmailConfigRepositoryImpl), "*"),
		wire.Struct(new(versions.VersionService), "*"),
		wire.Struct(new(setup.Server), "*"),
		wire.Struct(new(maintainers.UserServiceImpl), "*"),
		wire.Struct(new(apps.AppServiceImpl), "*"),
		wire.Struct(new(u.AuthHelperImpl)),
		wire.Struct(new(u.DatabaseUtilsImpl)),
		wire.Struct(new(u.BytesSignerImpl)),
		wire.Bind(new(u.BytesSigner), new(*u.BytesSignerImpl)),
		wire.Bind(new(u.DatabaseSnapshotRepository), new(*u.DatabaseSnapshotRepositoryImpl)),
		wire.Bind(new(u.DatabaseUtils), new(*u.DatabaseUtilsImpl)),
		wire.Bind(new(u.AuthHelper), new(*u.AuthHelperImpl)),
		wire.Bind(new(apps.AppRepository), new(*apps.AppRepositoryImpl)),
		wire.Bind(new(versions.VersionRepository), new(*versions.VersionRepositoryImpl)),
		wire.Bind(new(maintainers.UserRepository), new(*maintainers.UserRepositoryImpl)),
		wire.Bind(new(maintainers.EmailConfigRepository), new(*maintainers.EmailConfigRepositoryImpl)),
		wire.Bind(new(maintainers.EmailService), new(*maintainers.EmailServiceImpl)),
	)
	return nil
}

func NewDatabaseProvider(
	pathProvider *tools.PathProviderImpl,
	databaseUtils u.DatabaseUtils,
) *tools.DatabaseProviderImpl {
	return &tools.DatabaseProviderImpl{
		Db:            nil,
		PathProvider:  pathProvider,
		DatabaseUtils: databaseUtils,
	}
}

func NewDatabaseSnapshotRepository(
	databaseProvider *tools.DatabaseProviderImpl,
	databaseUtils u.DatabaseUtils,
) *u.DatabaseSnapshotRepositoryImpl {
	return u.NewDatabaseSnapshotRepository(tools.DefaultDatabaseHost, databaseProvider, databaseUtils)
}

func NewMux() *http.ServeMux {
	return http.NewServeMux()
}

func NewEmailClient(config *tools.Config) u.EmailClient {
	if config.UseMailMockClient {
		return &maintainers.TestEmailClient{}
	}
	return &u.EmailClientImpl{}
}

func NewRegistrationCodeProvider(config *tools.Config, authHelper u.AuthHelper) maintainers.SecretGenerator {
	if config.UseSampleDataForTesting {
		return &maintainers.RegistrationCodeProviderMock{}
	}
	return &maintainers.RegistrationCodeProviderImpl{
		AuthHelper: authHelper,
	}
}

func NewVersionValidator() validation.VersionValidator {
	return validation.NewVersionValidator(true)
}
