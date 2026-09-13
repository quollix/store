package local

import (
	"testing"

	"qsc/tools"

	"github.com/quollix/common/assert"
)

type updateFetcherTestDependencies struct {
	updateFetcher      *UpdateFetcherImpl
	fileSystemOperator *FileSystemOperatorMock
	registryTagFetcher *RegistryTagFetcherMock
}

func setupUpdateFetcherTest(t *testing.T) *updateFetcherTestDependencies {
	fileSystemOperatorMock := NewFileSystemOperatorMock(t)
	registryTagFetcherMock := NewRegistryTagFetcherMock(t)
	updateFetcher := &UpdateFetcherImpl{
		FileSystemOperator: fileSystemOperatorMock,
		RegistryTagFetcher: registryTagFetcherMock,
	}
	return &updateFetcherTestDependencies{
		updateFetcher:      updateFetcher,
		fileSystemOperator: fileSystemOperatorMock,
		registryTagFetcher: registryTagFetcherMock,
	}
}

func TestUpdateFetcherImpl_FetchUpdateAndWriteToReport_AddsServiceUpdate(t *testing.T) {
	deps := setupUpdateFetcherTest(t)
	report := getEmptyAppUpdateReport("sampleapp")
	composeContent := []byte("compose")
	deps.fileSystemOperator.EXPECT().GetForcedImages().Return(map[string]string{}, nil)
	deps.fileSystemOperator.EXPECT().ParseServicesFromCompose(composeContent).Return([]tools.Service{{Name: "sampleapp", Image: "sample/sampleapp", Tag: "1.0.0"}}, nil)
	deps.registryTagFetcher.EXPECT().GetLatestTag("sample/sampleapp", "1.0.0").Return("1.0.1", true, nil)
	deps.registryTagFetcher.EXPECT().GetDigest("sample/sampleapp", "1.0.1").Return(testDigestA, nil)

	err := deps.updateFetcher.FetchUpdateAndWriteToReport(report, composeContent)

	assert.Nil(t, err)
	assert.Equal(t, []tools.ServiceUpdate{{
		ServiceName: "sampleapp",
		ImageName:   "sample/sampleapp",
		OldTag:      "1.0.0",
		NewTag:      "1.0.1",
		NewDigest:   testDigestA,
	}}, report.ServiceUpdates)
}

func TestUpdateFetcherImpl_FetchUpdateAndWriteToReport_KeepsReportEmptyWhenNoUpdatesExist(t *testing.T) {
	deps := setupUpdateFetcherTest(t)
	report := getEmptyAppUpdateReport("sampleapp")
	composeContent := []byte("compose")
	deps.fileSystemOperator.EXPECT().GetForcedImages().Return(map[string]string{}, nil)
	deps.fileSystemOperator.EXPECT().ParseServicesFromCompose(composeContent).Return([]tools.Service{{Name: "sampleapp", Image: "sample/sampleapp", Tag: "1.0.0"}}, nil)
	deps.registryTagFetcher.EXPECT().GetLatestTag("sample/sampleapp", "1.0.0").Return("", false, nil)

	err := deps.updateFetcher.FetchUpdateAndWriteToReport(report, composeContent)

	assert.Nil(t, err)
	assert.Equal(t, []tools.ServiceUpdate{}, report.ServiceUpdates)
}

func TestUpdateFetcherImpl_FetchUpdateAndWriteToReport_UsesForcedImageTag(t *testing.T) {
	deps := setupUpdateFetcherTest(t)
	report := getEmptyAppUpdateReport("sampleapp")
	composeContent := []byte("compose")
	deps.fileSystemOperator.EXPECT().GetForcedImages().Return(map[string]string{"postgres": "17.5-alpine"}, nil)
	deps.fileSystemOperator.EXPECT().ParseServicesFromCompose(composeContent).Return([]tools.Service{{Name: "database", Image: "postgres", Tag: "16-alpine", Digest: testDigestC}}, nil)
	deps.registryTagFetcher.EXPECT().GetDigest("postgres", "17.5-alpine").Return(testDigestD, nil)

	err := deps.updateFetcher.FetchUpdateAndWriteToReport(report, composeContent)

	assert.Nil(t, err)
	assert.Equal(t, []tools.ServiceUpdate{{
		ServiceName:    "database",
		ImageName:      "postgres",
		OldTag:         "16-alpine",
		NewTag:         "17.5-alpine",
		OldDigest:      testDigestC,
		NewDigest:      testDigestD,
		ForcedByConfig: true,
	}}, report.ServiceUpdates)
}
