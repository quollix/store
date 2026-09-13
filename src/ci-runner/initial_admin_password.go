package main

import (
	"github.com/quollix/common/bootstrap"
	"github.com/quollix/common/deploy"
	"github.com/quollix/common/store"
	u "github.com/quollix/common/utils"
)

func TestServerInitialAdminPassword() {
	tr.Log.TaskDescription("Testing generated initial admin password")
	defer tr.Cleanup()

	BuildLocalDockerImage()
	runInitialAdminPasswordTestOrExit()
}

func runInitialAdminPasswordTestOrExit() {
	if err := verifyInitialAdminPassword(); err != nil {
		tr.Log.Error("Generated initial admin password test failed: %v", err)
		tr.ExitWithError()
		return
	}

	tr.Log.Info("Generated initial admin password sign-in succeeded")
}

func verifyInitialAdminPassword() error {
	deploy.CleanupLocal(tr, "store")
	deploy.DeployLocal(tr, "store", storeLocalServiceYAMLForProfile(prodProfile, false, false))

	username, password, err := bootstrap.WaitForGeneratedInitialAdminCredentials(storeContainerName)
	if err != nil {
		return err
	}
	if password == testInitialAdminPassword {
		return u.Logger.NewError("generated password should not be 'password', but random generated", "actual", password)
	}

	client := &store.AppStoreClientImpl{
		Parent: u.ComponentClient{
			SetCookieHeader: true,
			RootUrl:         "http://127.0.0.1:8080",
		},
	}
	if err := client.Login(username, password); err != nil {
		return err
	}

	account, err := client.GetAccountDetails()
	if err != nil {
		return err
	}
	if account.Name != username || !account.IsAdmin {
		return u.Logger.NewError("generated-password sign-in returned unexpected user", "username", account.Name, "is_admin", account.IsAdmin)
	}

	return nil
}
