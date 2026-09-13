package tools

import "time"

var (
	SampleMaintainer = "sample"
	SampleApp        = "sampleapp"
	SampleVersion    = "0.0.1"
	SampleEmail      = "sample@sample.com"
	SamplePassword   = "password"
)

type User struct {
	Id                  int
	Name                string
	Email               string
	HashedPassword      string
	PublicKeyRaw        []byte
	HashedCookieValue   *string
	ExpirationDate      *time.Time
	SetupTokenHash      *string
	SetupTokenExpiresAt *time.Time
	UsedSpaceInBytes    int64
	StorageLimitInBytes int64
	IsAdmin             bool
}

type App struct {
	AppId        int
	MaintainerId int
	Name         string
}
