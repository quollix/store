package local

import (
	"qsc/tools"

	u "github.com/quollix/common/utils"
)

type UpdateFetcher interface {
	FetchUpdateAndWriteToReport(report *tools.AppUpdateReport, composeContent []byte) error
}

type UpdateFetcherImpl struct {
	FileSystemOperator FileSystemOperator
	RegistryTagFetcher RegistryTagFetcher
}

func (a *UpdateFetcherImpl) FetchUpdateAndWriteToReport(report *tools.AppUpdateReport, composeContent []byte) error {
	forcedImages, err := a.FileSystemOperator.GetForcedImages()
	if err != nil {
		return err
	}
	services, err := a.FileSystemOperator.ParseServicesFromCompose(composeContent)
	if err != nil {
		return err
	}

	serviceUpdates, err := a.fetchUpdatesForServices(services, forcedImages)
	if err != nil {
		return err
	}
	report.ServiceUpdates = serviceUpdates
	return nil
}

func (a *UpdateFetcherImpl) fetchUpdatesForServices(services []tools.Service, forcedImages map[string]string) ([]tools.ServiceUpdate, error) {
	collectedUpdates := make([]tools.ServiceUpdate, 0, len(services))
	for _, service := range services {
		update, ok, err := a.fetchUpdatesForService(service, forcedImages)
		if err != nil {
			return nil, err
		}
		if ok {
			collectedUpdates = append(collectedUpdates, update)
		}
	}
	return collectedUpdates, nil
}

func (a *UpdateFetcherImpl) fetchUpdatesForService(service tools.Service, forcedImages map[string]string) (tools.ServiceUpdate, bool, error) {
	if forcedTag, ok := forcedImages[service.Image]; ok {
		if forcedTag == service.Tag {
			return tools.ServiceUpdate{}, false, nil
		}
		digest, err := a.RegistryTagFetcher.GetDigest(service.Image, forcedTag)
		if err != nil {
			return tools.ServiceUpdate{}, false, err
		}

		u.Logger.Info(
			"using forced tag for service",
			tools.ServiceNameField, service.Name,
			tools.OldServiceTagField, service.Tag,
			tools.NewerServiceTagField, forcedTag,
		)

		return tools.ServiceUpdate{
			ServiceName:    service.Name,
			ImageName:      service.Image,
			OldTag:         service.Tag,
			NewTag:         forcedTag,
			OldDigest:      service.Digest,
			NewDigest:      digest,
			ForcedByConfig: true,
		}, true, nil
	}

	latestTagFromRegistry, ok, err := a.RegistryTagFetcher.GetLatestTag(service.Image, service.Tag)
	if err != nil {
		return tools.ServiceUpdate{}, false, err
	}
	if !ok {
		return tools.ServiceUpdate{}, false, nil
	}
	digest, err := a.RegistryTagFetcher.GetDigest(service.Image, latestTagFromRegistry)
	if err != nil {
		return tools.ServiceUpdate{}, false, err
	}

	u.Logger.Info(
		"found newer tag for service",
		tools.ServiceNameField, service.Name,
		tools.OldServiceTagField, service.Tag,
		tools.NewerServiceTagField, latestTagFromRegistry,
	)

	return tools.ServiceUpdate{
		ServiceName: service.Name,
		ImageName:   service.Image,
		OldTag:      service.Tag,
		NewTag:      latestTagFromRegistry,
		OldDigest:   service.Digest,
		NewDigest:   digest,
	}, true, nil
}
