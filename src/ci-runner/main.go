package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/quollix/common/bootstrap"
	"github.com/quollix/common/ci"
	"github.com/quollix/common/deploy"
	u "github.com/quollix/common/utils"
	"github.com/quollix/taskrunner"
	"github.com/spf13/cobra"
)

var (
	tr                   = taskrunner.GetTaskRunner()
	keepLocalStoreSetup  bool
	keepLocalDeploySetup bool
	seedLocalSampleData  bool

	srcDir = getAbsoluteParentDir()

	ciRunnerDir = srcDir + "/ci-runner"

	serverDir       = srcDir + "/server"
	serverDockerDir = serverDir + "/docker"
	serverTestsDir  = serverDir + "/tests"

	clientDir      = srcDir + "/client"
	clientTestsDir = clientDir + "/tests"
)

func getAbsoluteParentDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return filepath.Dir(wd)
}

func main() {
	tr.EnableAbortForKeystrokeControlPlusC()
	tr.Config.CleanupFunc = func() {
		if keepLocalStoreSetup {
			tr.Log.Info("Skipping docker cleanup because keep flag is enabled")
			return
		}
		tr.Log.Info("Cleaning docker artifacts")
		deploy.CleanupLocal(tr, "store")
	}

	rootCmd := &cobra.Command{
		Use:   "ci-runner",
		Short: "local build, test, and deployment runner for Store",
	}

	testCmd.AddCommand(testAllCmd, testBuildCmd, testReleaseCmd)
	buildCmd.AddCommand(buildAllCmd)
	serverCmd.AddCommand(serverBuildCmd, serverTestCmd)
	serverTestCmd.AddCommand(serverTestUnitCmd, serverTestIntegrationCmd, serverTestComponentCmd, serverTestInitialAdminPasswordCmd, serverTestAllCmd)
	clientCmd.AddCommand(clientBuildCmd, clientTestCmd)
	clientTestCmd.AddCommand(clientTestUnitCmd, clientTestIntegrationCmd, clientTestTerminalCmd, clientTestManualCmd, clientTestAllCmd)
	clientTestCmd.PersistentFlags().BoolVarP(&keepLocalStoreSetup, "keep", "k", false, "Keep the local store test setup running after terminal tests")
	deployCmd.Flags().BoolVarP(&keepLocalDeploySetup, "keep", "k", false, "Keep the existing local store database")
	deployCmd.Flags().BoolVarP(&seedLocalSampleData, "sample-data", "s", false, "Seed deterministic local sample data")
	rootCmd.AddCommand(testCmd, releaseCmd, deployCmd, buildCmd, serverCmd, clientCmd, ci.NewCommonCmd(tr, srcDir, "store"))
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		tr.Log.Error("Error during execution: %s", err.Error())
		tr.ExitWithError()
	}
	tr.Log.Info("final exit code: 0")
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy current version to localhost",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if seedLocalSampleData && keepLocalDeploySetup {
			tr.Log.Error("sample data requires a fresh local deployment; use either --sample-data or --keep, not both")
			tr.ExitWithError()
		}
		BuildLocalDockerImage()
		if !keepLocalDeploySetup {
			deploy.CleanupLocal(tr, "store")
		}
		deploy.DeployLocal(tr, "store", storeLocalServiceYAMLForProfile(testProfile, seedLocalSampleData, true))
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "run tests",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var testAllCmd = &cobra.Command{
	Use:   "all",
	Short: "run all tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestAll()
	},
}

var testBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "compile all Go modules without running tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestBuild()
	},
}

var testReleaseCmd = &cobra.Command{
	Use:   "release",
	Short: "run release tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestRelease()
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build commands",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var buildAllCmd = &cobra.Command{
	Use:   "all",
	Short: "build server and client",
	Run: func(cmd *cobra.Command, args []string) {
		BuildAll()
	},
}

var releaseCmd = &cobra.Command{
	Use:   "release <tag>",
	Short: "build and publish the production Docker image",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ReleaseDockerImage(args[0])
	},
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "server module commands",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var serverBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "build server",
	Run: func(cmd *cobra.Command, args []string) {
		BuildServer()
	},
}

var serverTestCmd = &cobra.Command{
	Use:   "test",
	Short: "run server tests",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var serverTestUnitCmd = &cobra.Command{
	Use:   "unit",
	Short: "run server unit tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestServerUnit()
	},
}

var serverTestIntegrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "run server integration tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestServerIntegration()
	},
}

var serverTestComponentCmd = &cobra.Command{
	Use:   "component",
	Short: "run server component tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestServerComponent()
	},
}

var serverTestInitialAdminPasswordCmd = &cobra.Command{
	Use:   "initial-admin-password",
	Short: "verify generated initial admin password sign-in",
	Run: func(cmd *cobra.Command, args []string) {
		TestServerInitialAdminPassword()
	},
}

var serverTestAllCmd = &cobra.Command{
	Use:   "all",
	Short: "run all server tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestServerAll()
	},
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "client module commands",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

var clientBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "build client",
	Run: func(cmd *cobra.Command, args []string) {
		BuildClient()
	},
}

var clientTestCmd = &cobra.Command{
	Use:   "test",
	Short: "run client tests",
	Run: func(cmd *cobra.Command, args []string) {
		u.RunAndLogIfError(cmd.Help)
	},
}

func storeLocalServiceYAMLForProfile(profile string, sampleData bool, enableInitialAdminEnv bool) string {
	initialAdminEnv := ""
	if enableInitialAdminEnv {
		initialAdminEnv = fmt.Sprintf(`    %s: %s
    %s: %s
`, bootstrap.InitialAdminNameEnvVar, testInitialAdminName, bootstrap.InitialAdminPasswordEnvVar, testInitialAdminPassword)
	}

	return fmt.Sprintf(`
store:
  image: quollix/store:local
  container_name: quollix_store_store
  environment:
    PROFILE: %s
    SAMPLE_DATA: "%t"
%s
  ports:
    - "127.0.0.1:8080:8080"
  depends_on:
    - postgres
  networks:
    - quollix_store
`, profile, sampleData, initialAdminEnv)
}

var clientTestUnitCmd = &cobra.Command{
	Use:   "unit",
	Short: "run client unit tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestClientUnit()
	},
}

var clientTestIntegrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "run client integration tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestClientIntegration()
	},
}

var clientTestTerminalCmd = &cobra.Command{
	Use:   "terminal",
	Short: "run client terminal tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestClientTerminal()
	},
}

var clientTestManualCmd = &cobra.Command{
	Use:   "manual [run-filter]",
	Short: "run client manual tests",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runFilter := ""
		if len(args) == 1 {
			runFilter = args[0]
		}
		TestClientManual(runFilter)
	},
}

var clientTestAllCmd = &cobra.Command{
	Use:   "all",
	Short: "run all client tests",
	Run: func(cmd *cobra.Command, args []string) {
		TestClientAll()
	},
}
