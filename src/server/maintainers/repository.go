package maintainers

import (
	"crypto/ed25519"
	"database/sql"
	"errors"
	"server/tools"
	"time"

	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	selectUserQuery = `SELECT 
		maintainer_id, 
		maintainer_name, 
		email, 
		hashed_password, 
		ssh_public_key,
		hashed_cookie,
		expiration_date, 
		setup_token_hash,
		setup_token_expiration_date,
		used_space_in_bytes,
		storage_limit_in_bytes,
		is_admin
	FROM maintainers WHERE `
	userIdSqlField      = "maintainer_id"
	userNameIdField     = "maintainer_name"
	hashedCookieField   = "hashed_cookie"
	setupTokenHashField = "setup_token_hash"
)

var NotEnoughSpacePrefix = "not enough space"

type UserRepository interface {
	CreateUser(username, hashedPassword, email string, publicKeyRaw []byte, storageLimitInBytes int64, isAdmin bool) error
	CreateUserWithSetupToken(username, email string, publicKeyRaw, publicKeySignature []byte, storageLimitInBytes int64, setupTokenHash string, setupTokenExpirationDate time.Time) error
	GetUserViaHashedCookie(hashedCookieValue string) (*tools.User, error)
	GetUserById(userId int) (*tools.User, error)
	GetUserByName(user string) (*tools.User, error)
	GetUserBySetupTokenHash(setupTokenHash string) (*tools.User, error)
	GetMaintainerPublicKeyRecord(name string) (*store.MaintainerPublicKeyRecord, error)
	GetAllEmails() ([]string, error)
	DoesUserExist(user string) (bool, error)
	DoesEmailExist(email string) (bool, error)
	DoesPublicKeyRawExist(publicKeyRaw []byte) (bool, error)
	DeleteUser(userId int) error
	ChangePassword(userId int, newPassword string) error
	Logout(userId int) error
	SetInitialPassword(userId int, hashedPassword string) error

	UpdateUser(*tools.User) error

	DoesAdminAccountExist() (bool, error)
}

type UserRepositoryImpl struct {
	DatabaseProvider *tools.DatabaseProviderImpl
}

func (r *UserRepositoryImpl) GetUserById(userId int) (*tools.User, error) {
	return r.getUserBy(userIdSqlField, userId)
}

func (r *UserRepositoryImpl) getUserBy(field string, arg any) (*tools.User, error) {
	switch field {
	case userIdSqlField, userNameIdField, hashedCookieField, setupTokenHashField:
	default:
		return nil, u.Logger.NewError("unsupported field", tools.SqlQueryField, field)
	}

	var user tools.User
	err := r.DatabaseProvider.GetDb().QueryRow(selectUserQuery+field+" = $1", arg).Scan(
		&user.Id,
		&user.Name,
		&user.Email,
		&user.HashedPassword,
		&user.PublicKeyRaw,
		&user.HashedCookieValue,
		&user.ExpirationDate,
		&user.SetupTokenHash,
		&user.SetupTokenExpiresAt,
		&user.UsedSpaceInBytes,
		&user.StorageLimitInBytes,
		&user.IsAdmin,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetUserByName(userName string) (*tools.User, error) {
	usr, err := r.getUserBy(userNameIdField, userName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, u.Logger.NewError(IncorrectUsernameOrPasswordError)
	}
	if err != nil {
		return nil, err
	}
	return usr, nil
}

func (r *UserRepositoryImpl) GetUserViaHashedCookie(hashedCookieValue string) (*tools.User, error) {
	usr, err := r.getUserBy(hashedCookieField, hashedCookieValue)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, u.Logger.NewError(CookieNotFoundError)
	}
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	return usr, nil
}

func (r *UserRepositoryImpl) GetUserBySetupTokenHash(setupTokenHash string) (*tools.User, error) {
	usr, err := r.getUserBy(setupTokenHashField, setupTokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, u.Logger.NewError(RegistrationCodeNotFoundError)
	}
	if err != nil {
		return nil, err
	}
	return usr, nil
}

func (r *UserRepositoryImpl) GetAllEmails() ([]string, error) {
	rows, err := r.DatabaseProvider.GetDb().Query("SELECT email FROM maintainers ORDER BY maintainer_id")
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	defer u.Close(rows)

	var emails []string
	for rows.Next() {
		var email string
		if err = rows.Scan(&email); err != nil {
			return nil, u.Logger.NewError(err.Error())
		}
		emails = append(emails, email)
	}
	if err = rows.Err(); err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	return emails, nil
}

func (r *UserRepositoryImpl) UpdateUser(user *tools.User) error {
	_, err := r.DatabaseProvider.GetDb().Exec(
		`UPDATE maintainers 
		 SET maintainer_name = $1,
		     email = $2,
		     hashed_password = $3,
		     ssh_public_key = $4,
		     hashed_cookie = $5,
		     expiration_date = $6,
		     setup_token_hash = $7,
		     setup_token_expiration_date = $8,
		     used_space_in_bytes = $9,
		     storage_limit_in_bytes = $10
		 WHERE maintainer_id = $11`,
		user.Name,
		user.Email,
		user.HashedPassword,
		user.PublicKeyRaw,
		user.HashedCookieValue,
		user.ExpirationDate,
		user.SetupTokenHash,
		user.SetupTokenExpiresAt,
		user.UsedSpaceInBytes,
		user.StorageLimitInBytes,
		user.Id,
	)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *UserRepositoryImpl) DoesUserExist(user string) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow("SELECT EXISTS(SELECT 1 FROM maintainers WHERE maintainer_name = $1)", user).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *UserRepositoryImpl) CreateUser(username, hashedPassword, email string, publicKeyRaw []byte, storageLimitInBytes int64, isAdmin bool) error {
	_, err := r.DatabaseProvider.GetDb().Exec("INSERT INTO maintainers (maintainer_name, email, hashed_password, ssh_public_key, used_space_in_bytes, storage_limit_in_bytes, is_admin) VALUES ($1, $2, $3, $4, $5, $6, $7)", username, email, hashedPassword, publicKeyRaw, 0, storageLimitInBytes, isAdmin)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *UserRepositoryImpl) CreateUserWithSetupToken(username, email string, publicKeyRaw, publicKeySignature []byte, storageLimitInBytes int64, setupTokenHash string, setupTokenExpirationDate time.Time) error {
	_, err := r.DatabaseProvider.GetDb().Exec(
		`INSERT INTO maintainers
		 (maintainer_name, email, hashed_password, ssh_public_key, maintainer_public_key_signature, used_space_in_bytes, storage_limit_in_bytes, is_admin, setup_token_hash, setup_token_expiration_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8, $9)`,
		username,
		email,
		"",
		publicKeyRaw,
		publicKeySignature,
		0,
		storageLimitInBytes,
		setupTokenHash,
		setupTokenExpirationDate,
	)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *UserRepositoryImpl) GetMaintainerPublicKeyRecord(name string) (*store.MaintainerPublicKeyRecord, error) {
	var record store.MaintainerPublicKeyRecord
	err := r.DatabaseProvider.GetDb().QueryRow(
		`SELECT maintainer_name, ssh_public_key, maintainer_public_key_signature
		 FROM maintainers
		 WHERE maintainer_name = $1 AND is_admin = false`,
		name,
	).Scan(&record.Maintainer, &record.PublicKeyRaw, &record.PublicKeySignature)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, u.Logger.NewError(MaintainerNotFoundError)
	}
	if err != nil {
		return nil, u.Logger.NewError(err.Error())
	}
	if len(record.PublicKeySignature) != ed25519.SignatureSize {
		return nil, u.Logger.NewError(MaintainerPublicKeySignatureNotFoundError)
	}
	return &record, nil
}

func (r *UserRepositoryImpl) DeleteUser(userId int) error {
	_, err := r.DatabaseProvider.GetDb().Exec("DELETE FROM maintainers WHERE maintainer_id = $1", userId)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *UserRepositoryImpl) ChangePassword(userId int, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}

	_, err = r.DatabaseProvider.GetDb().Exec("UPDATE maintainers SET hashed_password = $1 WHERE maintainer_id = $2", hashedPassword, userId)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}

	return nil
}

func (r *UserRepositoryImpl) SetInitialPassword(userId int, hashedPassword string) error {
	_, err := r.DatabaseProvider.GetDb().Exec(
		`UPDATE maintainers
		 SET hashed_password = $1,
		     setup_token_hash = NULL,
		     setup_token_expiration_date = NULL
		 WHERE maintainer_id = $2`,
		hashedPassword,
		userId,
	)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *UserRepositoryImpl) Logout(userId int) error {
	_, err := r.DatabaseProvider.GetDb().Exec("UPDATE maintainers SET hashed_cookie = $1, expiration_date = $2 WHERE maintainer_id = $3", nil, nil, userId)
	if err != nil {
		return u.Logger.NewError(err.Error())
	}
	return nil
}

func (r *UserRepositoryImpl) DoesEmailExist(email string) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow("SELECT EXISTS(SELECT 1 FROM maintainers WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *UserRepositoryImpl) DoesPublicKeyRawExist(publicKeyRaw []byte) (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow("SELECT EXISTS(SELECT 1 FROM maintainers WHERE ssh_public_key = $1)", publicKeyRaw).Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}

func (r *UserRepositoryImpl) DoesAdminAccountExist() (bool, error) {
	var exists bool
	err := r.DatabaseProvider.GetDb().QueryRow("SELECT EXISTS(SELECT 1 FROM maintainers WHERE is_admin = true)").Scan(&exists)
	if err != nil {
		return false, u.Logger.NewError(err.Error())
	}
	return exists, nil
}
