package tools

type ServiceUpdate struct {
	ServiceName    string
	ImageName      string
	OldTag         string
	NewTag         string
	OldDigest      string
	NewDigest      string
	ForcedByConfig bool
}

type Service struct {
	Name   string
	Image  string
	Tag    string
	Digest string
}

type DockerHubAuth struct {
	Username string `yaml:"username,omitempty"`
	Token    string `yaml:"token,omitempty"`
}

var SampleDockerHubAuth = DockerHubAuth{ // #nosec G101 -- sample auth used only for local test/dev fixtures.
	Username: "quollix-test",
	Token:    "quollix-test-docker-hub-token",
}

type FullUpdateReport struct {
	WasSuccessful    bool              `yaml:"was_successful"`
	AppUpdateReports []AppUpdateReport `yaml:"app_update_reports"`
}

type AppUpdateReport struct {
	AppName        string          `yaml:"app_name"`
	ServiceUpdates []ServiceUpdate `yaml:"service_updates,omitempty"`
	ErrorMessage   string          `yaml:"error_message,omitempty"`
}

func (a *AppUpdateReport) IsSuccessfulSoFar() bool {
	return a.ErrorMessage == ""
}

func (a *AppUpdateReport) HasUpdates() bool {
	return len(a.ServiceUpdates) > 0
}
