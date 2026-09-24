package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/quollix/common/deploy"
	u "github.com/quollix/common/utils"
)

var (
	dockerImage = "quollix/store:"
	testProfile = "TEST"
	prodProfile = "PROD"

	storeContainerName    = "quollix_store_store"
	databaseContainerName = "quollix_store_postgres"
)

const (
	testInitialAdminName     = "quollix"
	testInitialAdminPassword = "password"
)

func TestAll() {
	TestBuild()
	TestServerUnit()
	TestServerIntegration()
	TestClientUnit()
	TestClientIntegration()
	TestServerComponent()
	TestServerInitialAdminPassword()
	TestClientTerminal()
}

func TestBuild() {
	tr.Log.TaskDescription("Building all Go modules without running tests")
	u.BuildWholeGoProjectByBuildTag(tr, serverDir)
	u.BuildWholeGoProjectByBuildTag(tr, clientDir)
	u.BuildWholeGoProjectByBuildTag(tr, ciRunnerDir)
}

func TestRelease() {
	tr.Log.TaskDescription("No release tests configured")
}

func BuildClient() {
	tr.Log.TaskDescription("Building client")
	defer tr.Cleanup()
	u.BuildWholeGoProject(tr, clientDir)
}

func TestClientUnit() {
	tr.Log.TaskDescription("Running client unit tests")
	defer tr.Cleanup()
	BuildClient()
	u.GoTest(clientDir).Run(tr)
}

func TestClientIntegration() {
	tr.Log.TaskDescription("Running client integration tests")
	defer tr.Cleanup()
	u.GoTest(clientDir).Tag("integration").Run(tr)
}

func TestClientTerminal() {
	tr.Log.TaskDescription("Running client terminal tests")
	if keepLocalStoreSetup {
		running, err := (&u.DockerCliWrapperImpl{}).IsContainerRunning(storeContainerName)
		if err != nil {
			tr.Log.Error("failed to inspect local store setup: %v", err)
			tr.ExitWithError()
		}
		if !running {
			BuildLocalDockerImage()
			deploy.DeployLocal(tr, "store", storeLocalServiceYAMLForProfile(testProfile, true, true))
		}
	} else {
		BuildLocalDockerImage()
		deploy.CleanupLocal(tr, "store")
		deploy.DeployLocal(tr, "store", storeLocalServiceYAMLForProfile(testProfile, true, true))
		tr.Cmd().AsDaemon("store logs").Run("docker logs -f quollix_store_store")
	}
	defer tr.Cleanup()
	u.GoTest(clientTestsDir).Tag("terminal").Env("PROFILE", testProfile).Run(tr)
}

func TestClientManual(runFilter string) {
	tr.Log.TaskDescription("Running client manual tests")
	defer tr.Cleanup()
	command := u.GoTest(clientDir).Tag("manual")
	if runFilter != "" {
		command.Filter(runFilter)
	}
	command.Run(tr)
}

func TestClientAll() {
	TestClientUnit()
	TestClientIntegration()
	TestClientTerminal()
}

func BuildServer() {
	tr.Log.TaskDescription("Building server")
	defer tr.Cleanup()
	u.BuildWholeGoProject(tr, serverDir)
}

func BuildAll() {
	BuildServer()
	BuildClient()
}

func TestServerUnit() {
	tr.Log.TaskDescription("Running server unit tests")
	defer tr.Cleanup()
	BuildServer()
	u.GoTest(serverDir).Run(tr)
}

func TestServerIntegration() {
	tr.Log.TaskDescription("Running server integration tests")
	defer tr.Cleanup()
	tempDir := startLocalPostgres("store")
	defer u.RemoveDir(tempDir)
	u.GoTest(serverTestsDir).Tag("integration").Run(tr)
}

func TestServerComponent() {
	tr.Log.TaskDescription("Running server component tests")
	defer tr.Cleanup()
	BuildLocalDockerImage()
	runComponentTestWithProfile(testProfile, "component")
	runComponentTestWithProfile(prodProfile, "production")
}

func TestServerAll() {
	TestServerUnit()
	TestServerIntegration()
	TestServerComponent()
	TestServerInitialAdminPassword()
}

func runComponentTestWithProfile(profile string, testTag string) {
	deploy.CleanupLocal(tr, "store")
	deploy.DeployLocal(tr, "store", storeLocalServiceYAMLForProfile(profile, false, true))
	tr.Cmd().AsDaemon("store logs").Run("docker logs -f %s", storeContainerName)
	u.GoTest(serverTestsDir).Tag(testTag).Run(tr)
}

func BuildLocalDockerImage() {
	BuildDockerImage("local")
}

func BuildDockerImage(tag string) {
	image := dockerImage + tag
	tr.Cmd().
		Dir(serverDir).
		Env("CGO_ENABLED", "0").
		Env("GOOS", "linux").
		Env("GOARCH", "amd64").
		Run("go build -installsuffix cgo")
	command := fmt.Sprintf("docker build -t %s -f %s/Dockerfile .", image, serverDockerDir)
	tr.Cmd().Dir(serverDir).Run("%s", command)
}

func ReleaseDockerImage(tag string) {
	image := dockerImage + tag
	BuildDockerImage(tag)
	tr.Cmd().Dir(serverDir).Run("docker push %s", image)
	tr.Log.Info("Published Docker image: %s", image)
}

func startLocalPostgres(appName string) string {
	tempDir, err := os.MkdirTemp("", appName+"-postgres-")
	if err != nil {
		tr.Log.Error("failed to create temp dir for postgres compose: %v", err)
		tr.ExitWithError()
	}

	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	content := fmt.Sprintf(`services:
  postgres:
    image: postgres:17.2
    container_name: %s
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_HOST_AUTH_METHOD=trust
      - POSTGRES_DB=postgres
    ports:
      - "127.0.0.1:5432:5432"
    tmpfs:
      - /var/lib/postgresql/data
`, databaseContainerName)
	if err := os.WriteFile(composeFile, []byte(content), 0o644); err != nil { // #nosec G306 -- temp compose file for local docker compose invocation
		u.RemoveDir(tempDir)
		tr.Log.Error("failed to write temp postgres compose: %v", err)
		tr.ExitWithError()
	}

	tr.Cmd().Dir(tempDir).Run("docker compose -f %s up -d postgres", composeFile)

	return tempDir
}
