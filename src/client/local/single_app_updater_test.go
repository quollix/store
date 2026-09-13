package local

import (
	"testing"

	u "github.com/quollix/common/utils"
	"qsc/tools"

	"github.com/quollix/common/assert"
	"github.com/stretchr/testify/mock"
)

type singleAppUpdaterTestDependencies struct {
	singleAppUpdater      *SingleAppUpdaterImpl
	fileSystemOperator    *FileSystemOperatorMock
	updateFetcher         *UpdateFetcherMock
	composeContentUpdater *ComposeContentUpdaterMock
}

func setupSingleAppUpdaterTest(t *testing.T) *singleAppUpdaterTestDependencies {
	fileSystemOperatorMock := NewFileSystemOperatorMock(t)
	updateFetcherMock := NewUpdateFetcherMock(t)
	composeContentUpdaterMock := NewComposeContentUpdaterMock(t)
	return &singleAppUpdaterTestDependencies{
		singleAppUpdater: &SingleAppUpdaterImpl{
			FileSystemOperator:    fileSystemOperatorMock,
			AppUpdater:            updateFetcherMock,
			ComposeContentUpdater: composeContentUpdaterMock,
		},
		fileSystemOperator:    fileSystemOperatorMock,
		updateFetcher:         updateFetcherMock,
		composeContentUpdater: composeContentUpdaterMock,
	}
}

func TestSingleAppUpdaterImpl_ConductUpdateForSingleApp_WritesUpdatedComposeFile(t *testing.T) {
	deps := setupSingleAppUpdaterTest(t)
	sourceComposePath := tools.GetAppComposePath(tools.AppsDir, tools.SampleAppName)
	composeContent := []byte("old compose")
	updatedComposeContent := []byte("new compose")
	serviceUpdates := []tools.ServiceUpdate{{ServiceName: "web", ImageName: "nginx", OldTag: "1.0", NewTag: "1.1"}}
	deps.fileSystemOperator.EXPECT().GetDockerComposeFileContent(sourceComposePath).Return(composeContent, nil)
	deps.updateFetcher.EXPECT().FetchUpdateAndWriteToReport(mock.Anything, composeContent).Run(func(report *tools.AppUpdateReport, _ []byte) {
		report.ServiceUpdates = serviceUpdates
	}).Return(nil)
	deps.composeContentUpdater.EXPECT().UpdateComposeTags(composeContent, serviceUpdates).Return(updatedComposeContent, nil)
	deps.fileSystemOperator.EXPECT().WriteDockerComposeFileContent(sourceComposePath, updatedComposeContent).Return(nil)

	report := deps.singleAppUpdater.ConductUpdateForSingleApp(tools.SampleAppName)

	assert.Equal(t, tools.SampleAppName, report.AppName)
	assert.Equal(t, "", report.ErrorMessage)
	assert.Equal(t, serviceUpdates, report.ServiceUpdates)
}

func TestSingleAppUpdaterImpl_ConductUpdateForSingleApp_DoesNotWriteWhenThereAreNoUpdates(t *testing.T) {
	deps := setupSingleAppUpdaterTest(t)
	sourceComposePath := tools.GetAppComposePath(tools.AppsDir, tools.SampleAppName)
	composeContent := []byte("compose")
	deps.fileSystemOperator.EXPECT().GetDockerComposeFileContent(sourceComposePath).Return(composeContent, nil)
	deps.updateFetcher.EXPECT().FetchUpdateAndWriteToReport(mock.Anything, composeContent).Return(nil)

	report := deps.singleAppUpdater.ConductUpdateForSingleApp(tools.SampleAppName)

	assert.Equal(t, tools.SampleAppName, report.AppName)
	assert.Equal(t, "", report.ErrorMessage)
	assert.Equal(t, []tools.ServiceUpdate(nil), report.ServiceUpdates)
}

func TestSingleAppUpdaterImpl_ConductUpdateForSingleApp_StoresErrorMessage(t *testing.T) {
	deps := setupSingleAppUpdaterTest(t)
	sourceComposePath := tools.GetAppComposePath(tools.AppsDir, tools.SampleAppName)
	deps.fileSystemOperator.EXPECT().GetDockerComposeFileContent(sourceComposePath).Return(nil, u.Logger.NewError("read failed"))

	report := deps.singleAppUpdater.ConductUpdateForSingleApp(tools.SampleAppName)

	assert.Equal(t, tools.SampleAppName, report.AppName)
	assert.Equal(t, "read failed", report.ErrorMessage)
}
