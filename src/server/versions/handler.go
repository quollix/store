package versions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"server/apps"
	"server/maintainers"
	"server/tools"

	"github.com/quollix/common/store"
	"github.com/quollix/common/validation"

	u "github.com/quollix/common/utils"
)

var expectedVersionUploadErrors = u.MapOf(
	maintainers.MaximumStorageExceededError,
	maintainers.InvalidPublicKeyError,
	VersionAlreadyExist,
	store.VersionContentAlreadyExists,
	VersionUploadTimestampTooOld,
	VersionUploadTimestampInFuture,
	VersionUploadTimestampNotNewer,
	VersionContentContainsCarriageReturn,
	InvalidVersionSignature,
)

type VersionsHandler struct {
	VersionRepo    VersionRepository
	AppRepo        apps.AppRepository
	UserRepo       maintainers.UserRepository
	VersionService *VersionService
	UserService    *maintainers.UserServiceImpl
	AppService     *apps.AppServiceImpl
	Validator      validation.VersionValidator
}

func (v *VersionsHandler) VersionsUploadHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, tools.VersionUploadLimitInBytes)
	defer u.Close(r.Body)
	var versionUpload store.VersionUploadDto
	err := json.NewDecoder(r.Body).Decode(&versionUpload)
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		msg := fmt.Sprintf("version content too large, the limit is %d MB", tools.VersionUploadLimitInBytes/tools.OneMegaByteInBytes)
		u.Logger.Info(msg, tools.UserField, user)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}

	err = v.Validator.Validate(versionUpload.Content, user.Name, versionUpload.AppName)
	if err != nil {
		u.WriteResponseErrorAlways(w, err)
		return
	}

	createdVersion, err := v.VersionService.UploadVersion(user, versionUpload)
	if err != nil {
		u.WriteResponseError(w, expectedVersionUploadErrors, err)
		return
	}
	u.SendJsonResponse(w, createdVersion)
}

func (v *VersionsHandler) VersionDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	versionID, ok := validation.ReadBody[store.VersionID](w, r)
	if !ok {
		return
	}
	err := v.VersionService.DeleteVersionByIDWithChecks(user, versionID.VersionId)
	if err != nil {
		u.WriteResponseError(w, tools.DoesNotExistErrorMap, err, tools.UserField, user)
	}
}

func (v *VersionsHandler) VersionMigrationCheckpointHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	request, ok := validation.ReadBody[store.VersionMigrationCheckpointRequest](w, r)
	if !ok {
		return
	}
	err := v.VersionService.SetMigrationCheckpointByIDWithChecks(user, request.VersionId, request.IsMigrationCheckpoint)
	if err != nil {
		u.WriteResponseError(w, tools.DoesNotExistErrorMap, err, tools.UserField, user)
	}
}

func (v *VersionsHandler) GetVersionsHandler(w http.ResponseWriter, r *http.Request) {
	appTree, ok := validation.ReadBody[store.AppTree](w, r)
	if !ok {
		return
	}
	versionsList, err := v.VersionService.ListVersions(appTree.Maintainer, appTree.AppName)
	if err != nil {
		u.WriteResponseError(w, tools.DoesNotExistErrorMap, err)
		return
	}
	u.SendJsonResponse(w, versionsList)
}

// Deprecated: use VersionDownloadByIDHandler.
func (v *VersionsHandler) VersionDownloadHandler(w http.ResponseWriter, r *http.Request) {
	versionTree, ok := validation.ReadBody[store.VersionTree](w, r)
	if !ok {
		return
	}
	versionInfo, err := v.VersionRepo.GetVersion(versionTree.Maintainer, versionTree.AppName, versionTree.VersionName)
	if err != nil {
		u.WriteResponseError(w, tools.DoesNotExistErrorMap, err)
		return
	}
	u.SendJsonResponse(w, versionInfo)
}

func (v *VersionsHandler) VersionDownloadByIDHandler(w http.ResponseWriter, r *http.Request) {
	versionID, ok := validation.ReadBody[store.VersionID](w, r)
	if !ok {
		return
	}
	versionInfo, err := v.VersionRepo.GetVersionByID(versionID.VersionId)
	if err != nil {
		u.WriteResponseError(w, tools.DoesNotExistErrorMap, err)
		return
	}
	u.SendJsonResponse(w, versionInfo)
}

func (v *VersionsHandler) DownloadNextVersionForUpdateHandler(w http.ResponseWriter, r *http.Request) {
	request, ok := validation.ReadBody[store.NextVersionForUpdateRequest](w, r)
	if !ok {
		return
	}
	response, err := v.VersionService.DownloadNextVersionForUpdate(*request)
	if err != nil {
		u.WriteResponseError(w, tools.DoesNotExistErrorMap, err)
		return
	}
	u.SendJsonResponse(w, response)
}
