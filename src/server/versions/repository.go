package versions

import (
	"database/sql"
	"errors"
	"time"

	"server/apps"
	"server/maintainers"
	"server/tools"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

type VersionContentHash [32]byte

type VersionRepository interface {
	DoesVersionNameExistByNames(userName, appName, version string) (bool, error)
	DoesVersionNameExistByAppID(appId int, version string) (bool, error)
	DoesVersionContentHashExist(contentHash VersionContentHash) (bool, error)
	DoesVersionBelongToMaintainer(userId int, versionId int) (bool, error)
	GetLatestVersionCreationTimestampByAppID(appId int) (*time.Time, error)
	CreateVersion(appId int, version string, creationTimestamp time.Time, data []byte, contentHash VersionContentHash, signature []byte) (*store.CreatedVersionResponse, error)
	DeleteVersionByID(versionId int) error
	SetMigrationCheckpointByID(versionId int, isMigrationCheckpoint bool) error
	ListVersionsOfApp(userName, appName string) ([]store.LeanVersionDto, error)
	// Deprecated: use GetVersionByID.
	GetVersion(userName, appName, version string) (*store.Version, error)
	GetVersionByID(versionId int) (*store.Version, error)
	GetNextVersionForUpdate(userName, appName string, currentVersionCreationTimestamp time.Time) (*store.Version, error)
}

type VersionRepositoryImpl struct {
	DatabaseProvider *tools.DatabaseProviderImpl
	UserRepo         maintainers.UserRepository
	AppRepo          apps.AppRepository
}

func (r *VersionRepositoryImpl) DoesVersionNameExistByNames(userName, appName, version string) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow(`SELECT EXISTS(
		SELECT 1 
		FROM versions
		JOIN apps ON versions.app_id = apps.app_id
		JOIN maintainers ON apps.maintainer_id = maintainers.maintainer_id
		WHERE maintainers.maintainer_name = $1 AND apps.app_name = $2 AND versions.version_name = $3
	)`, userName, appName, version).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *VersionRepositoryImpl) DoesVersionNameExistByAppID(appId int, version string) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM versions
			WHERE app_id = $1 AND version_name = $2
		)`,
		appId, version,
	).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *VersionRepositoryImpl) DoesVersionContentHashExist(contentHash VersionContentHash) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM versions
			WHERE content_sha256 = $1
		)`,
		contentHash[:],
	).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *VersionRepositoryImpl) DoesVersionBelongToMaintainer(userId int, versionId int) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM versions
			JOIN apps ON versions.app_id = apps.app_id
			WHERE apps.maintainer_id = $1 AND versions.version_id = $2
		)`,
		userId,
		versionId,
	).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *VersionRepositoryImpl) GetLatestVersionCreationTimestampByAppID(appId int) (*time.Time, error) {
	var creationTimestamp time.Time
	err := r.DatabaseProvider.GetDb().QueryRow(`
		SELECT creation_timestamp
		FROM versions
		WHERE app_id = $1
		ORDER BY creation_timestamp DESC
		LIMIT 1
	`, appId).Scan(&creationTimestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	creationTimestamp = creationTimestamp.UTC()
	return &creationTimestamp, nil
}

// Deprecated: use GetVersionByID.
func (r *VersionRepositoryImpl) GetVersion(userName, appName, versionName string) (*store.Version, error) {
	var version store.Version
	err := r.DatabaseProvider.GetDb().QueryRow(`
		SELECT versions.version_id, maintainers.maintainer_name, apps.app_name, versions.version_name, versions.data, versions.signature, maintainers.ssh_public_key, versions.creation_timestamp, versions.is_migration_checkpoint
		FROM versions
		JOIN apps ON versions.app_id = apps.app_id
		JOIN maintainers ON apps.maintainer_id = maintainers.maintainer_id
		WHERE maintainers.maintainer_name = $1 AND apps.app_name = $2 AND versions.version_name = $3
		ORDER BY versions.creation_timestamp DESC
		LIMIT 1
	`, userName, appName, versionName).Scan(&version.VersionId, &version.Maintainer, &version.AppName, &version.VersionName, &version.Content, &version.Signature, &version.MaintainerPublicKeyRaw, &version.VersionCreationTimestamp, &version.IsMigrationCheckpoint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, u.Logger.NewError(tools.DoesNotExistError)
	}
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}

	version.VersionCreationTimestamp = version.VersionCreationTimestamp.UTC()
	return &version, nil
}

func (r *VersionRepositoryImpl) GetVersionByID(versionId int) (*store.Version, error) {
	var version store.Version
	err := r.DatabaseProvider.GetDb().QueryRow(`
		WITH downloaded_version AS (
			UPDATE versions
			SET download_count = download_count + 1
			WHERE version_id = $1
			RETURNING version_id, app_id, version_name, data, signature, creation_timestamp, is_migration_checkpoint
		)
		SELECT downloaded_version.version_id, maintainers.maintainer_name, apps.app_name, downloaded_version.version_name, downloaded_version.data, downloaded_version.signature, maintainers.ssh_public_key, downloaded_version.creation_timestamp, downloaded_version.is_migration_checkpoint
		FROM downloaded_version
		JOIN apps ON downloaded_version.app_id = apps.app_id
		JOIN maintainers ON apps.maintainer_id = maintainers.maintainer_id
	`, versionId).Scan(&version.VersionId, &version.Maintainer, &version.AppName, &version.VersionName, &version.Content, &version.Signature, &version.MaintainerPublicKeyRaw, &version.VersionCreationTimestamp, &version.IsMigrationCheckpoint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, u.Logger.NewError(tools.DoesNotExistError)
	}
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}

	version.VersionCreationTimestamp = version.VersionCreationTimestamp.UTC()
	return &version, nil
}

func (r *VersionRepositoryImpl) GetNextVersionForUpdate(userName, appName string, currentVersionCreationTimestamp time.Time) (*store.Version, error) {
	var version store.Version
	err := r.DatabaseProvider.GetDb().QueryRow(`
		WITH target_app AS (
			SELECT apps.app_id
			FROM apps
			JOIN maintainers ON apps.maintainer_id = maintainers.maintainer_id
			WHERE maintainers.maintainer_name = $1
			  AND apps.app_name = $2
		), next_migration_checkpoint AS (
			SELECT versions.version_id, 1 AS priority
			FROM versions
			JOIN target_app ON versions.app_id = target_app.app_id
			WHERE versions.creation_timestamp > $3
			  AND versions.is_migration_checkpoint = TRUE
			ORDER BY versions.creation_timestamp ASC
			LIMIT 1
		), latest_version AS (
			SELECT versions.version_id, 0 AS priority
			FROM versions
			JOIN target_app ON versions.app_id = target_app.app_id
			WHERE versions.creation_timestamp > $3
			ORDER BY versions.creation_timestamp DESC
			LIMIT 1
		), selected_version AS (
			SELECT version_id
			FROM (
				SELECT * FROM next_migration_checkpoint
				UNION ALL
				SELECT * FROM latest_version
			) candidates
			ORDER BY priority DESC
			LIMIT 1
		), downloaded_version AS (
			UPDATE versions
			SET download_count = download_count + 1
			FROM selected_version
			WHERE versions.version_id = selected_version.version_id
			RETURNING versions.version_id, versions.app_id, versions.version_name, versions.data, versions.signature, versions.creation_timestamp, versions.is_migration_checkpoint
		)
		SELECT downloaded_version.version_id, maintainers.maintainer_name, apps.app_name, downloaded_version.version_name, downloaded_version.data, downloaded_version.signature, maintainers.ssh_public_key, downloaded_version.creation_timestamp, downloaded_version.is_migration_checkpoint
		FROM downloaded_version
		JOIN apps ON downloaded_version.app_id = apps.app_id
		JOIN maintainers ON apps.maintainer_id = maintainers.maintainer_id
	`, userName, appName, currentVersionCreationTimestamp.UTC()).Scan(&version.VersionId, &version.Maintainer, &version.AppName, &version.VersionName, &version.Content, &version.Signature, &version.MaintainerPublicKeyRaw, &version.VersionCreationTimestamp, &version.IsMigrationCheckpoint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}

	version.VersionCreationTimestamp = version.VersionCreationTimestamp.UTC()
	return &version, nil
}

func (r *VersionRepositoryImpl) CreateVersion(appId int, version string, creationTimestamp time.Time, data []byte, contentHash VersionContentHash, signature []byte) (*store.CreatedVersionResponse, error) {
	app, err := r.AppRepo.GetAppById(appId)
	if err != nil {
		return nil, err
	}

	tx, err := r.DatabaseProvider.GetDb().Begin()
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			u.Logger.Error(err)
		}
	}()

	dataSize := int64(len(data))
	updateResult, err := tx.Exec(
		`UPDATE maintainers
		 SET used_space_in_bytes = used_space_in_bytes + $1
		 WHERE maintainer_id = $2
		   AND used_space_in_bytes + $1 <= storage_limit_in_bytes`,
		dataSize,
		app.MaintainerId,
	)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}

	rowsAffected, err := updateResult.RowsAffected()
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	if rowsAffected == 0 {
		return nil, u.Logger.NewError(maintainers.MaximumStorageExceededError)
	}

	var createdVersion store.CreatedVersionResponse
	err = tx.QueryRow("INSERT INTO versions (app_id, version_name, creation_timestamp, data, content_sha256, signature) VALUES ($1, $2, $3, $4, $5, $6) RETURNING app_id, version_id", appId, version, creationTimestamp.UTC(), data, contentHash[:], signature).Scan(&createdVersion.AppId, &createdVersion.VersionId)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}

	return &createdVersion, tx.Commit()
}

func (r *VersionRepositoryImpl) DeleteVersionByID(versionId int) error {
	result, err := r.DatabaseProvider.GetDb().Exec(`
		WITH deleted AS (
			DELETE FROM versions v
			USING apps a
			WHERE v.app_id = a.app_id
			  AND v.version_id = $1
			RETURNING length(v.data) AS size_in_bytes, a.maintainer_id
		)
		UPDATE maintainers m
		SET used_space_in_bytes = m.used_space_in_bytes - d.size_in_bytes
		FROM deleted d
		WHERE m.maintainer_id = d.maintainer_id
	`, versionId)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	if rowsAffected == 0 {
		return u.Logger.NewError(tools.DoesNotExistError)
	}

	return nil
}

func (r *VersionRepositoryImpl) SetMigrationCheckpointByID(versionId int, isMigrationCheckpoint bool) error {
	result, err := r.DatabaseProvider.GetDb().Exec(`
		UPDATE versions
		SET is_migration_checkpoint = $1
		WHERE version_id = $2
	`, isMigrationCheckpoint, versionId)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	if rowsAffected == 0 {
		return u.Logger.NewError(tools.DoesNotExistError)
	}
	return nil
}

func (r *VersionRepositoryImpl) ListVersionsOfApp(userName, appName string) ([]store.LeanVersionDto, error) {
	rows, err := r.DatabaseProvider.GetDb().Query(`
		SELECT v.version_id,
		       v.version_name,
		       v.creation_timestamp,
		       length(v.data) AS size_in_bytes,
		       v.is_migration_checkpoint,
		       v.download_count
		FROM versions v
		JOIN apps a ON v.app_id = a.app_id
		JOIN maintainers u ON a.maintainer_id = u.maintainer_id
		WHERE u.maintainer_name = $1 AND a.app_name = $2
		ORDER BY v.creation_timestamp DESC
	`, userName, appName)
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	defer u.Close(rows)

	var versions []store.LeanVersionDto
	for rows.Next() {
		var dto store.LeanVersionDto
		if err = rows.Scan(&dto.VersionId, &dto.Name, &dto.CreationTimestamp, &dto.SizeInBytes, &dto.IsMigrationCheckpoint, &dto.DownloadCount); err != nil {
			return nil, u.Logger.NewError(err.Error())
		}
		versions = append(versions, dto)
	}
	if err = rows.Err(); err != nil {
		return nil, u.Logger.NewError(err.Error())
	}

	return versions, nil
}
