package apps

import (
	"database/sql"
	"errors"
	"fmt"
	"server/maintainers"
	"server/tools"
	"time"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

var (
	officialMaintainerName           = "quollix"
	queryFilterForOfficialMaintainer = fmt.Sprintf(" AND u.maintainer_name = '%s'", officialMaintainerName)
)

type AppRepository interface {
	DoesAppIdExist(appId int) (bool, error)
	CreateApp(userId int, app string) error
	DeleteApp(userId int, appName string) error
	GetAppList(userId int) ([]string, error)
	SearchForApps(searchRequest store.SearchRequest) ([]store.AppWithLatestVersion, error)
	GetAppById(appId int) (*tools.App, error)
	GetAppByName(userId int, appName string) (*tools.App, error)
	SumUpBytesOfAllAppVersions(appID int) (int64, error)
	GetNumberOfApps(userId int) (int, error)

	DoesAppExistByMaintainerName(userName, app string) (bool, error)
	DoesAppExistByMaintainerID(userId int, appName string) (bool, error)
}

type AppRepositoryImpl struct {
	DatabaseProvider *tools.DatabaseProviderImpl
	UserRepo         maintainers.UserRepository
}

func (r *AppRepositoryImpl) DoesAppExistByMaintainerName(userName, appName string) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT EXISTS(
			SELECT 1 
			FROM apps 
			JOIN maintainers ON apps.maintainer_id = maintainers.maintainer_id 
			WHERE maintainers.maintainer_name = $1 AND apps.app_name = $2
		)`, userName, appName).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *AppRepositoryImpl) DoesAppExistByMaintainerID(userId int, appName string) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM apps
			WHERE maintainer_id = $1 AND app_name = $2
		)`,
		userId, appName,
	).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *AppRepositoryImpl) GetAppByName(userId int, appName string) (*tools.App, error) {
	var app tools.App
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT app_id, maintainer_id, app_name
		 FROM apps
		WHERE maintainer_id = $1 AND app_name = $2`,
		userId, appName,
	).Scan(&app.AppId, &app.MaintainerId, &app.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, u.Logger.NewError(tools.DoesNotExistError)
	}
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	return &app, nil
}

func (r *AppRepositoryImpl) GetAppById(appId int) (*tools.App, error) {
	var app tools.App
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT app_id, maintainer_id, app_name
		 FROM apps
		WHERE app_id = $1`,
		appId,
	).Scan(&app.AppId, &app.MaintainerId, &app.Name)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	return &app, nil
}

func (r *AppRepositoryImpl) CreateApp(userId int, app string) error {
	_, err := r.DatabaseProvider.GetDb().Exec(`INSERT INTO apps (maintainer_id, app_name) VALUES ($1, $2)`, userId, app)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *AppRepositoryImpl) DoesAppIdExist(appId int) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow("SELECT EXISTS(SELECT 1 FROM apps WHERE app_id = $1)", appId).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *AppRepositoryImpl) DeleteApp(userId int, appName string) error {
	_, err := r.DatabaseProvider.GetDb().Exec(`DELETE FROM apps WHERE maintainer_id = $1 AND app_name = $2`, userId, appName)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *AppRepositoryImpl) SumUpBytesOfAllAppVersions(appID int) (int64, error) {
	var totalSize sql.NullInt64
	err := r.DatabaseProvider.GetDb().QueryRow("SELECT SUM(LENGTH(data)) FROM versions WHERE app_id = $1", appID).Scan(&totalSize)
	if err != nil {
		return 0, u.Logger.NewError(err.Error())
	}
	if !totalSize.Valid {
		return 0, nil
	}
	return totalSize.Int64, nil
}

func (r *AppRepositoryImpl) SearchForApps(request store.SearchRequest) ([]store.AppWithLatestVersion, error) {
	var apps []store.AppWithLatestVersion

	query := `
		SELECT u.maintainer_name, a.app_name, v.version_id, v.version_name, v.creation_timestamp
		FROM maintainers u
		JOIN apps a ON u.maintainer_id = a.maintainer_id
		-- Trigram indexes on maintainer_name/app_name accelerate these ILIKE filters
		JOIN LATERAL (
		  SELECT version_id, version_name, creation_timestamp
		  FROM versions
		  WHERE app_id = a.app_id
		  ORDER BY creation_timestamp DESC
		  LIMIT 1
		) v ON TRUE
		WHERE u.maintainer_name ILIKE $1 AND a.app_name ILIKE $2
`
	if !request.ShowUnofficialApps {
		// Cheap equality filter; avoids string concat in the main SQL text
		query += queryFilterForOfficialMaintainer
	}
	query += `
		ORDER BY
		  -- Rank exact (case-insensitive) maintainer/app matches first before LIMIT
		  (LOWER(u.maintainer_name) = LOWER($3)) DESC,
		  (LOWER(a.app_name)  = LOWER($4)) DESC,
		  -- Cheap deterministic tie-breaker
		  a.app_id
		LIMIT 100
`

	rows, err := r.DatabaseProvider.GetDb().Query(
		query,
		"%"+request.MaintainerSearchTerm+"%",
		"%"+request.AppSearchTerm+"%",
		request.MaintainerSearchTerm,
		request.AppSearchTerm,
	)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	defer u.Close(rows)

	for rows.Next() {
		var maintainer, appName, versionName string
		var latestVersionId int
		var latestVersionCreationTimestamp time.Time
		if err = rows.Scan(&maintainer, &appName, &latestVersionId, &versionName, &latestVersionCreationTimestamp); err != nil {
			u.Logger.Error(err)
			continue
		}
		apps = append(apps, store.AppWithLatestVersion{
			Maintainer:                     maintainer,
			AppName:                        appName,
			LatestVersionId:                latestVersionId,
			LatestVersionName:              versionName,
			LatestVersionCreationTimestamp: latestVersionCreationTimestamp.UTC(),
		})
	}
	if err = rows.Err(); err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	return apps, nil
}

func (r *AppRepositoryImpl) GetAppList(userId int) ([]string, error) {
	rows, err := r.DatabaseProvider.GetDb().Query("SELECT app_name FROM apps WHERE maintainer_id = $1", userId)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	defer u.Close(rows)

	var apps []string
	for rows.Next() {
		var app string
		if err = rows.Scan(&app); err != nil {
			return nil, u.Logger.NewError(err.Error())
		}
		apps = append(apps, app)
	}

	if err = rows.Err(); err != nil {
		return nil, u.Logger.NewError(err.Error())
	}

	return apps, nil
}

func (r *AppRepositoryImpl) GetNumberOfApps(userId int) (int, error) {
	var count int
	err := r.DatabaseProvider.GetDb().QueryRow("SELECT COUNT(*) FROM apps WHERE maintainer_id = $1", userId).Scan(&count)
	if err != nil {
		return 0, u.Logger.NewError(err.Error())
	}
	return count, nil
}
