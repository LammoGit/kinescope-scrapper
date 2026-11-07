package services

type ProvisioningService struct {
}

func NewProvisioningService() (*ProvisioningService, error) {
	return &ProvisioningService{}, nil
}
