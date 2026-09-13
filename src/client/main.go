package main

import (
	"fmt"
	"os"
	"qsc/commands"
	"qsc/di"
	"qsc/tools"

	"github.com/quollix/common/utils"
	"github.com/quollix/deepstack"
	"github.com/quollix/taskrunner"
)

func main() {
	taskRunner := taskrunner.GetTaskRunner()
	taskRunner.EnableAbortForKeystrokeControlPlusC()

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		utils.Logger.Error(err, "result", "failed to get user config dir")
		os.Exit(1)
	}
	profile := os.Getenv("PROFILE")
	globalConfig := tools.InitGlobalConfig(profile, userConfigDir)
	commands.Dependencies = di.BuildDependencyGraph(globalConfig)
	commands.InitializeCobraCommands(commands.Dependencies)

	if err := commands.Dependencies.SessionManager.ApplySessionFromConfig(); err != nil {
		utils.Logger.Debug(err, "result", "no existing session found, skipping loading session")
	}

	if _, err := commands.RootCmd.ExecuteC(); err != nil {
		fmt.Println(renderCliError(err))
		os.Exit(1)
	}
}

func renderCliError(err error) string {
	deepStackError, ok := err.(*deepstack.DeepStackError)
	if !ok {
		return utils.ExtractError(err)
	}

	responseBody, ok := deepStackError.Context["response_body"].(string)
	if ok && responseBody != "" {
		return responseBody
	}

	return deepStackError.Message
}
