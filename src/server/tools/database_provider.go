package tools

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	u "github.com/quollix/common/utils"
)

const (
	DefaultDatabaseHost = "quollix_store_postgres"
)

type DatabaseProviderImpl struct {
	Db            *sql.DB
	PathProvider  *PathProviderImpl
	DatabaseUtils u.DatabaseUtils
	Host          string
}

func (d *DatabaseProviderImpl) GetDb() *sql.DB {
	return d.Db
}

func (d *DatabaseProviderImpl) GetDB() *sql.DB {
	return d.GetDb()
}

func (d *DatabaseProviderImpl) Connect() error {
	db, err := d.DatabaseUtils.WaitForPostgresDb(d.Host, u.DefaultPostgresPort, u.PostgresApplicationDatabase)
	if err != nil {
		return err
	}

	d.Db = db
	return nil
}

func (d *DatabaseProviderImpl) InitializeDatabase() error {
	if d.Host == "" {
		d.Host = DefaultDatabaseHost
	}

	err := d.DatabaseUtils.EnsureDatabaseExists(d.Host, u.DefaultPostgresPort, u.PostgresApplicationDatabase)
	if err != nil {
		return err
	}

	d.Db, err = d.DatabaseUtils.WaitForPostgresDb(d.Host, u.DefaultPostgresPort, u.PostgresApplicationDatabase)
	if err != nil {
		return err
	}

	return d.DatabaseUtils.RunMigrations(d.PathProvider.GetMigrationsDir(), d.Host, u.DefaultPostgresPort, u.PostgresApplicationDatabase)
}
