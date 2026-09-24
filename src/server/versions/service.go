package versions

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"time"

	"server/apps"
	"server/maintainers"
	"server/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

var (
	VersionAlreadyExist                  = "version already exists"
	VersionUploadTimestampTooOld         = "version upload timestamp is too old"
	VersionUploadTimestampInFuture       = "version upload timestamp is in the future"
	VersionUploadTimestampNotNewer       = "version upload timestamp is not newer than the latest app version"
	VersionContentContainsCarriageReturn = "version content contains carriage return characters"
	InvalidVersionSignature              = "invalid version signature"
)

type VersionService struct {
	VersionRepo VersionRepository
	UserService *maintainers.UserServiceImpl
	AppRepo     apps.AppRepository
	AppService  *apps.AppServiceImpl
	UserRepo    maintainers.UserRepository
	BytesSigner u.BytesSigner
}

func (s *VersionService) DeleteVersionByIDWithChecks(user *tools.User, versionId int) error {
	doesOwnVersion, err := s.VersionRepo.DoesVersionBelongToMaintainer(user.Id, versionId)
	if err != nil {
		return err
	}
	if !doesOwnVersion {
		return u.Logger.NewError(tools.DoesNotExistError)
	}
	return s.VersionRepo.DeleteVersionByID(versionId)
}

func (s *VersionService) SetMigrationCheckpointByIDWithChecks(user *tools.User, versionId int, isMigrationCheckpoint bool) error {
	doesOwnVersion, err := s.VersionRepo.DoesVersionBelongToMaintainer(user.Id, versionId)
	if err != nil {
		return err
	}
	if !doesOwnVersion {
		return u.Logger.NewError(tools.DoesNotExistError)
	}
	return s.VersionRepo.SetMigrationCheckpointByID(versionId, isMigrationCheckpoint)
}

func (s *VersionService) UploadVersion(user *tools.User, versionUpload store.VersionUploadDto) (*store.CreatedVersionResponse, error) {
	err := validation.ValidateStruct(versionUpload)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(versionUpload.Content, []byte("\r")) {
		return nil, u.Logger.NewError(VersionContentContainsCarriageReturn)
	}

	creationTimestamp := versionUpload.CreationTimestamp.UTC()
	now := time.Now().UTC()
	if creationTimestamp.After(now) {
		return nil, u.Logger.NewError(VersionUploadTimestampInFuture)
	}
	if creationTimestamp.Before(now.Add(-time.Hour)) {
		return nil, u.Logger.NewError(VersionUploadTimestampTooOld)
	}

	doesExist, err := s.AppRepo.DoesAppExistByMaintainerID(user.Id, versionUpload.AppName)
	if err != nil {
		return nil, err
	}

	if !doesExist {
		if err := s.AppRepo.CreateApp(user.Id, versionUpload.AppName); err != nil {
			return nil, err
		}
	}

	app, err := s.AppRepo.GetAppByName(user.Id, versionUpload.AppName)
	if err != nil {
		return nil, err
	}

	latestCreationTimestamp, err := s.VersionRepo.GetLatestVersionCreationTimestampByAppID(app.AppId)
	if err != nil {
		return nil, err
	}
	if latestCreationTimestamp != nil && !creationTimestamp.After(*latestCreationTimestamp) {
		return nil, u.Logger.NewError(VersionUploadTimestampNotNewer)
	}

	contentHash := VersionContentHash(sha256.Sum256(versionUpload.Content))
	doesContentHashExist, err := s.VersionRepo.DoesVersionContentHashExist(contentHash)
	if err != nil {
		return nil, err
	}
	if doesContentHashExist {
		return nil, u.Logger.NewError(store.VersionContentAlreadyExists)
	}

	payloadBytes, err := (&store.VersionSigningCodecImpl{}).EncodeVersion(&store.Version{
		Maintainer:               user.Name,
		AppName:                  versionUpload.AppName,
		VersionName:              versionUpload.Version,
		Content:                  versionUpload.Content,
		VersionCreationTimestamp: creationTimestamp,
	})
	if err != nil {
		return nil, err
	}
	if !s.BytesSigner.VerifyBytes(ed25519.PublicKey(user.PublicKeyRaw), payloadBytes, versionUpload.Signature) {
		return nil, u.Logger.NewError(InvalidVersionSignature)
	}

	createdVersion, err := s.VersionRepo.CreateVersion(app.AppId, versionUpload.Version, creationTimestamp, versionUpload.Content, contentHash, versionUpload.Signature)
	if err != nil {
		return nil, err
	}
	return createdVersion, nil
}

func (s *VersionService) ListVersions(userName, appName string) ([]store.LeanVersionDto, error) {
	if err := s.AppService.AssertAppIsPresent(userName, appName); err != nil {
		return nil, err
	}
	versionsList, err := s.VersionRepo.ListVersionsOfApp(userName, appName)
	if err != nil {
		return nil, err
	}
	return versionsList, nil
}

func (s *VersionService) DownloadNextVersionForUpdate(request store.NextVersionForUpdateRequest) (*store.NextVersionForUpdateResponse, error) {
	if err := validation.ValidateStruct(request); err != nil {
		return nil, err
	}
	if err := s.AppService.AssertAppIsPresent(request.Maintainer, request.AppName); err != nil {
		return nil, err
	}

	version, err := s.VersionRepo.GetNextVersionForUpdate(request.Maintainer, request.AppName, request.CurrentVersionCreationTimestamp)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return &store.NextVersionForUpdateResponse{UpdateAvailable: false}, nil
	}
	return &store.NextVersionForUpdateResponse{UpdateAvailable: true, Version: version}, nil
}

func (s *VersionService) AssertVersionIsPresent(userName, appName, versionName string) error {
	exists, err := s.VersionRepo.DoesVersionNameExistByNames(userName, appName, versionName)
	if err != nil {
		return err
	}
	if !exists {
		return u.Logger.NewError(tools.DoesNotExistError)
	}
	return nil
}
