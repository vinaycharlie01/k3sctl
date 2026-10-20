package binprovider

import "context"

type existingProvider struct {
	host Host
}

func NewExisting(host Host) *existingProvider {
	return &existingProvider{host: host}
}

func (p *existingProvider) NeedsUpgrade(_ context.Context) (bool, error) {
	return false, nil
}

func (p *existingProvider) IsUpload() bool {
	return false
}

func (p *existingProvider) Stage(_ context.Context) error {
	return nil
}

func (p *existingProvider) CleanUp(_ context.Context) error {
	return nil
}
