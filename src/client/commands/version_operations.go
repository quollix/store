package commands

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"qsc/remote"
	"qsc/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

type versionUploadConfig struct {
	Maintainer           string
	PrivateKeyPath       string
	PrivateKeyPassphrase string
	SelectedApps         []string
}

type versionUploadRequest struct {
	Maintainer           string
	VersionName          string
	AppPath              string
	PrivateKeyPath       string
	PrivateKeyPassphrase string
}

type versionUploadResult struct {
	AppName string
	Message string
}

func uploadVersionsFromAppsDirectory(config versionUploadConfig) ([]versionUploadResult, error) {
	appNames, err := Dependencies.AppSelector.SelectApps(config.SelectedApps)
	if err != nil {
		return nil, err
	}
	if len(appNames) == 0 {
		return nil, u.Logger.NewError("no apps found")
	}

	results := make([]versionUploadResult, 0, len(appNames))
	didUploadFail := false
	for _, appName := range appNames {
		request, err := prepareVersionUploadRequest(config, appName)
		if err != nil {
			logVersionUploadError(appName, err)
			results = append(results, versionUploadResult{AppName: appName, Message: "upload failed"})
			didUploadFail = true
			continue
		}
		err = uploadVersion(*request)
		if isExistingVersionContentError(err) {
			results = append(results, versionUploadResult{AppName: appName, Message: "store already has this content"})
			continue
		}
		if err != nil {
			logVersionUploadError(appName, err)
			results = append(results, versionUploadResult{AppName: appName, Message: "upload failed"})
			didUploadFail = true
			continue
		}
		results = append(results, versionUploadResult{AppName: appName, Message: fmt.Sprintf("uploaded version %s", request.VersionName)})
	}
	if didUploadFail {
		return results, u.Logger.NewError("one or more app uploads failed")
	}
	return results, nil
}

func logVersionUploadError(appName string, err error) {
	u.Logger.Error(err, tools.AppField, appName, "result", "app upload failed")
}

func isExistingVersionContentError(err error) bool {
	responseErrorMessage, ok := u.ExtractResponseErrorMessage(err)
	return ok && responseErrorMessage == store.VersionContentAlreadyExists
}

func prepareVersionUploadRequest(config versionUploadConfig, appName string) (*versionUploadRequest, error) {
	localConfig, err := Dependencies.ConfigProvider.GetConfig()
	if err != nil {
		return nil, err
	}
	appsDirectory, err := localConfig.RequireAppsDirectory()
	if err != nil {
		return nil, err
	}
	appPath := tools.GetAppComposePath(appsDirectory, appName)
	appContent, err := Dependencies.OsWrapper.ReadFile(appPath)
	if err != nil {
		return nil, err
	}
	appContent = normalizeVersionUploadContent(appContent)
	if err := Dependencies.VersionValidator.Validate(appContent, config.Maintainer, appName); err != nil {
		return nil, err
	}
	versionName, err := extractMainServiceVersionName(appContent, appName)
	if err != nil {
		return nil, err
	}
	if err := validateUploadVersionName(appName, versionName); err != nil {
		return nil, err
	}
	return &versionUploadRequest{
		Maintainer:           config.Maintainer,
		VersionName:          versionName,
		AppPath:              appPath,
		PrivateKeyPath:       config.PrivateKeyPath,
		PrivateKeyPassphrase: config.PrivateKeyPassphrase,
	}, nil
}

func validateUploadVersionName(appName string, versionName string) error {
	return validation.ValidateStruct(store.VersionUploadDto{
		AppName: appName,
		Version: versionName,
	})
}

func extractMainServiceVersionName(appContent []byte, appName string) (string, error) {
	services, err := Dependencies.FileSystemOperator.ParseServicesFromCompose(appContent)
	if err != nil {
		return "", err
	}
	for _, service := range services {
		if service.Name != appName {
			continue
		}
		_, versionName, _ := tools.SplitTag(strings.TrimSpace(service.Tag))
		if versionName == "" {
			return "", u.Logger.NewError("main service image tag is empty", tools.AppField, appName)
		}
		return versionName, nil
	}
	return "", u.Logger.NewError(tools.MainServiceNotFoundError, tools.AppField, appName)
}

func uploadVersion(request versionUploadRequest) error {
	appName := tools.GetAppNameFromComposePath(request.AppPath)
	appContent, err := Dependencies.OsWrapper.ReadFile(request.AppPath)
	if err != nil {
		return err
	}
	appContent = normalizeVersionUploadContent(appContent)
	privateKeyOpenSSH, err := Dependencies.OsWrapper.ReadFile(request.PrivateKeyPath)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	creationTimestamp := Dependencies.OsWrapper.Now().UTC()
	signature, err := remote.SignVersionPayload(Dependencies.VersionSigningService, privateKeyOpenSSH, request.PrivateKeyPassphrase, request.Maintainer, appName, request.VersionName, creationTimestamp, appContent)
	if err != nil {
		return err
	}
	_, err = Dependencies.AppStoreClient.UploadVersionAndReturnCreatedVersion(appName, request.VersionName, creationTimestamp, appContent, signature)
	if err != nil {
		return err
	}
	return nil
}

func normalizeVersionUploadContent(content []byte) []byte {
	normalizedContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	normalizedContent = strings.ReplaceAll(normalizedContent, "\r", "\n")
	return []byte(normalizedContent)
}

func cloneLatestVersions(maintainer string, downloadPath string) (int, error) {
	folderExists, err := Dependencies.OsWrapper.DoesFileExist(downloadPath)
	if err != nil {
		return 0, err
	}
	if folderExists {
		if err := promptMaintainerToDeletionConfirmOperation(fmt.Sprintf("deletion of clone folder '%s'", downloadPath)); err != nil {
			return 0, err
		}
		if err := Dependencies.OsWrapper.RemoveAll(downloadPath); err != nil {
			return 0, err
		}
	}
	if err := Dependencies.OsWrapper.MkdirAll(downloadPath, 0o700); err != nil {
		return 0, err
	}
	return downloadLatestVersions(maintainer, downloadPath)
}

func downloadLatestVersions(maintainer string, downloadPath string) (int, error) {
	apps, err := Dependencies.AppStoreClient.ListOwnApps()
	if err != nil {
		return 0, err
	}
	sort.Strings(apps)

	cloneCount := 0
	for _, app := range apps {
		versions, err := Dependencies.AppStoreClient.ListVersions(maintainer, app)
		if err != nil {
			return 0, err
		}
		if len(versions) == 0 {
			continue
		}
		latestVersion := orderVersionsNewestFirst(versions)[0]
		if _, err := downloadVersionComposeFileByID(app, latestVersion.VersionId, downloadPath); err != nil {
			return 0, err
		}
		cloneCount++
	}
	return cloneCount, nil
}

func downloadVersionComposeFileByID(appName string, versionID int, downloadPath string) (string, error) {
	version, err := fetchVerifiedVersionByID(versionID)
	if err != nil {
		return "", err
	}
	appFilePath := filepath.Join(downloadPath, fmt.Sprintf("%s.yml", appName))
	if err := Dependencies.OsWrapper.WriteFile(appFilePath, version.Content, 0o600); err != nil {
		return "", err
	}
	return appFilePath, nil
}

func fetchVerifiedVersionContentByID(versionID int) ([]byte, error) {
	version, err := fetchVerifiedVersionByID(versionID)
	if err != nil {
		return nil, err
	}
	return version.Content, nil
}

func fetchVerifiedVersionByID(versionID int) (*store.Version, error) {
	version, err := Dependencies.AppStoreClient.DownloadVersionByID(versionID)
	if err != nil {
		return nil, err
	}
	if err := validateDownloadedVersion(version); err != nil {
		return nil, err
	}
	return version, nil
}

func validateDownloadedVersion(version *store.Version) error {
	if version == nil {
		return u.Logger.NewError("version must not be nil")
	}
	if err := Dependencies.VersionValidator.Validate(version.Content, version.Maintainer, version.AppName); err != nil {
		return err
	}
	maintainerPublicKey, err := resolveTrustedMaintainerPublicKey(version.Maintainer)
	if err != nil {
		return err
	}
	if !bytes.Equal(maintainerPublicKey, version.MaintainerPublicKeyRaw) {
		return u.Logger.NewError("downloaded maintainer public key does not match trusted public key record")
	}
	valid, err := Dependencies.VersionSigningService.VerifyVersionSignature(maintainerPublicKey, version)
	if err != nil {
		return err
	}
	if !valid {
		return u.Logger.NewError("invalid version signature")
	}
	return nil
}

func resolveTrustedMaintainerPublicKey(maintainer string) (ed25519.PublicKey, error) {
	if maintainer == u.OfficialMaintainer {
		return u.DecodeAuthorizedEd25519PublicKey([]byte(getAppStoreOfficialMaintainerPublicKeyOpenSSH()))
	}

	maintainerPublicKey, err := resolveCachedMaintainerPublicKey(maintainer)
	if err == nil {
		return maintainerPublicKey, nil
	}
	u.Logger.Warn("failed to load cached maintainer public key record, fetching from server", "maintainer", maintainer, "details", u.ExtractError(err))
	return fetchMaintainerPublicKeyFromServer(maintainer)
}

func fetchMaintainerPublicKeyFromServer(maintainer string) (ed25519.PublicKey, error) {
	record, err := Dependencies.AppStoreClient.GetMaintainerPublicKeyRecord(maintainer)
	if err != nil {
		return nil, err
	}
	if err := verifyMaintainerPublicKeyRecord(record); err != nil {
		return nil, err
	}
	if err := Dependencies.SessionManager.SaveMaintainerPublicKeyRecord(record); err != nil {
		return nil, err
	}
	return ed25519.PublicKey(record.PublicKeyRaw), nil
}

func resolveCachedMaintainerPublicKey(maintainer string) (ed25519.PublicKey, error) {
	record, err := Dependencies.SessionManager.LoadMaintainerPublicKeyRecord(maintainer)
	if err != nil {
		return nil, err
	}
	if err := verifyMaintainerPublicKeyRecord(record); err != nil {
		return nil, err
	}
	return ed25519.PublicKey(record.PublicKeyRaw), nil
}

func verifyMaintainerPublicKeyRecord(record *store.MaintainerPublicKeyRecord) error {
	adminPublicKey, err := u.DecodeAuthorizedEd25519PublicKey([]byte(getAppStoreOfficialMaintainerPublicKeyOpenSSH()))
	if err != nil {
		return err
	}
	valid, err := store.VerifyMaintainerPublicKeySignature(adminPublicKey, record)
	if err != nil {
		return err
	}
	if !valid {
		return u.Logger.NewError("invalid maintainer public key signature")
	}
	return nil
}

func getAppStoreOfficialMaintainerPublicKeyOpenSSH() string {
	if Dependencies.Config.UseLocalTestingOfficialMaintainerPublicKey {
		return u.LocalTestingPublicKeyOpenSSH
	}
	return store.AppStoreOfficialMaintainerPublicKeyOpenSSH
}

func orderVersionsNewestFirst(versions []store.LeanVersionDto) []store.LeanVersionDto {
	orderedVersions := append([]store.LeanVersionDto(nil), versions...)
	sort.SliceStable(orderedVersions, func(i, j int) bool {
		if orderedVersions[i].CreationTimestamp.Equal(orderedVersions[j].CreationTimestamp) {
			return orderedVersions[i].VersionId > orderedVersions[j].VersionId
		}
		return orderedVersions[i].CreationTimestamp.After(orderedVersions[j].CreationTimestamp)
	})
	return orderedVersions
}

func orderVersionsOldestFirst(versions []store.LeanVersionDto) []store.LeanVersionDto {
	orderedVersions := append([]store.LeanVersionDto(nil), versions...)
	sort.SliceStable(orderedVersions, func(i, j int) bool {
		if orderedVersions[i].CreationTimestamp.Equal(orderedVersions[j].CreationTimestamp) {
			return orderedVersions[i].VersionId < orderedVersions[j].VersionId
		}
		return orderedVersions[i].CreationTimestamp.Before(orderedVersions[j].CreationTimestamp)
	})
	return orderedVersions
}
