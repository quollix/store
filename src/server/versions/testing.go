package versions

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"server/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"github.com/quollix/common/validation"
)

func (v *VersionsHandler) AddIncompatibleVersionHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := tools.GetUserFromContextOrWriteError(w, r)
	if !ok {
		return
	}
	request, ok := validation.ReadBody[store.AddIncompatibleVersionRequest](w, r)
	if !ok {
		return
	}
	created, err := v.VersionService.AddIncompatibleVersion(user, *request)
	if err != nil {
		u.WriteResponseError(w, nil, err)
		return
	}
	u.SendJsonResponse(w, created)
}

func (s *VersionService) AddIncompatibleVersion(user *tools.User, request store.AddIncompatibleVersionRequest) (*store.CreatedVersionResponse, error) {
	exists, err := s.AppRepo.DoesAppExistByMaintainerID(user.Id, request.AppName)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := s.AppRepo.CreateApp(user.Id, request.AppName); err != nil {
			return nil, err
		}
	}
	app, err := s.AppRepo.GetAppByName(user.Id, request.AppName)
	if err != nil {
		return nil, err
	}
	timestamp := time.Now().UTC()
	content := []byte(fmt.Sprintf("# incompatible test fixture: %s/%s/%s at %s\nservices: [invalid", user.Name, request.AppName, request.Version, timestamp.Format(time.RFC3339Nano)))
	return s.VersionRepo.CreateVersion(app.AppId, request.Version, timestamp, content, VersionContentHash(sha256.Sum256(content)), []byte{})
}
