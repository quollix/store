//go:build wireinject
// +build wireinject

package di

import (
	"qsc/commands"
	"qsc/local"
	"qsc/remote"
	"qsc/tools"

	"github.com/google/wire"
	"github.com/quollix/common/store"
	"github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

func BuildDependencyGraph(config *tools.GlobalConfig) *commands.ClientDependencies {
	wire.Build(
		NewAppStoreClient,
		NewConfiguredRegistryTagFetcher,
		NewImageTagCache,
		NewDockerHubRegistry,
		NewVersionValidator,
		wire.Struct(new(commands.ClientDependencies), "*"),
		wire.Struct(new(remote.SessionManagerImpl), "*"),
		wire.Struct(new(remote.SigningKeyManagerImpl), "*"),
		wire.Struct(new(store.VersionSigningServiceImpl), "*"),
		wire.Struct(new(store.VersionSigningCodecImpl)),
		wire.Struct(new(local.FileSystemOperatorImpl), "*"),
		wire.Struct(new(local.OCIRegistryImpl)),
		wire.Struct(new(local.ConsistencyCheckerImpl), "*"),
		wire.Struct(new(local.ComposeContentUpdaterImpl), "*"),
		wire.Struct(new(local.ImageReferenceParserImpl), "*"),
		wire.Struct(new(local.UpdateHelperImpl), "*"),
		wire.Struct(new(local.UpdateFetcherImpl), "*"),
		wire.Struct(new(local.Updater), "*"),
		wire.Struct(new(local.SingleAppUpdaterImpl), "*"),
		wire.Struct(new(local.AppSelectorImpl), "*"),
		wire.Struct(new(local.TagSelectorImpl), "*"),
		wire.Struct(new(utils.OsWrapperImpl)),
		wire.Struct(new(utils.BytesSignerImpl)),
		wire.Bind(new(utils.OsWrapper), new(*utils.OsWrapperImpl)),
		wire.Bind(new(utils.BytesSigner), new(*utils.BytesSignerImpl)),
		wire.Bind(new(remote.SessionManager), new(*remote.SessionManagerImpl)),
		wire.Bind(new(remote.SigningKeyManager), new(*remote.SigningKeyManagerImpl)),
		wire.Bind(new(store.VersionSigningCodec), new(*store.VersionSigningCodecImpl)),
		wire.Bind(new(store.VersionSigningService), new(*store.VersionSigningServiceImpl)),
		wire.Bind(new(local.DockerHubAuthProvider), new(*remote.SessionManagerImpl)),
		wire.Bind(new(local.ConsistencyChecker), new(*local.ConsistencyCheckerImpl)),
		wire.Bind(new(local.FileSystemOperator), new(*local.FileSystemOperatorImpl)),
		wire.Bind(new(local.ComposeContentUpdater), new(*local.ComposeContentUpdaterImpl)),
		wire.Bind(new(local.ImageReferenceParser), new(*local.ImageReferenceParserImpl)),
		wire.Bind(new(local.UpdateHelper), new(*local.UpdateHelperImpl)),
		wire.Bind(new(local.UpdateFetcher), new(*local.UpdateFetcherImpl)),
		wire.Bind(new(local.SingleAppUpdater), new(*local.SingleAppUpdaterImpl)),
		wire.Bind(new(local.AppSelector), new(*local.AppSelectorImpl)),
		wire.Bind(new(local.TagSelector), new(*local.TagSelectorImpl)),
		wire.Bind(new(local.DockerHubRegistry), new(*local.DockerHubRegistryImpl)),
		wire.Bind(new(local.OCIRegistry), new(*local.OCIRegistryImpl)),
	)
	return nil
}

func NewAppStoreClient(config *tools.GlobalConfig) *store.AppStoreClientImpl {
	return &store.AppStoreClientImpl{
		Parent: utils.ComponentClient{
			SetCookieHeader:   true,
			RootUrl:           config.AppStoreRootURL,
			Origin:            "",
			VerifyCertificate: config.VerifyAppStoreCertificate,
			Cookie:            nil,
		},
	}
}

func NewVersionValidator() validation.VersionValidator {
	return validation.NewVersionValidator(false)
}

func NewDockerHubRegistry(authProvider local.DockerHubAuthProvider) *local.DockerHubRegistryImpl {
	return &local.DockerHubRegistryImpl{
		DockerHubAuthProvider: authProvider,
	}
}

func NewConfiguredRegistryTagFetcher(config *tools.GlobalConfig, dockerHubRegistry local.DockerHubRegistry, ociRegistry local.OCIRegistry, tagSelector local.TagSelector, imageTagCache local.ImageTagCache, imageReferenceParser local.ImageReferenceParser) local.RegistryTagFetcher {
	if config.UseTestRegistryTagFetcher {
		return &local.RegistryTagFetcherStubImpl{}
	}
	return &local.RegistryTagFetcherImpl{
		DockerHubRegistry:    dockerHubRegistry,
		OCIRegistry:          ociRegistry,
		TagSelector:          tagSelector,
		ImageTagCache:        imageTagCache,
		ImageReferenceParser: imageReferenceParser,
	}
}

func NewImageTagCache(config *tools.GlobalConfig, osWrapper utils.OsWrapper) local.ImageTagCache {
	if !config.EnableImageTagCache {
		return &local.NoOpImageTagCache{}
	}
	return &local.ImageTagCacheImpl{
		OsWrapper: osWrapper,
		FilePath:  config.ImageTagCachePath,
	}
}
