package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/fekuna/omnipos-customer-service/internal/customer/repository"
	"github.com/fekuna/omnipos-customer-service/internal/model"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UseCase interface {
	CreateCustomer(ctx context.Context, input *model.Customer) (*model.Customer, error)
	GetCustomer(ctx context.Context, id string) (*model.Customer, error)
	ListCustomers(ctx context.Context, merchantID string, page, pageSize int, search string) ([]*model.Customer, int, error)
	UpdateCustomer(ctx context.Context, input *model.Customer) (*model.Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
	AddLoyaltyPoints(ctx context.Context, customerID string, points int32) (int32, error)
}

type customerUseCase struct {
	repo   repository.Repository
	logger logger.ZapLogger
}

func NewCustomerUseCase(repo repository.Repository, logger logger.ZapLogger) UseCase {
	return &customerUseCase{repo: repo, logger: logger}
}

func (uc *customerUseCase) CreateCustomer(ctx context.Context, input *model.Customer) (*model.Customer, error) {
	// Generate ID if missing
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}
	// Initial Loyalty Points
	input.LoyaltyPoints = 0
	input.CreatedAt = time.Now()
	input.UpdatedAt = time.Now()

	if err := uc.repo.Create(ctx, input); err != nil {
		uc.logger.Error("Failed to create customer", zap.Error(err))
		return nil, err
	}
	return input, nil
}

func (uc *customerUseCase) GetCustomer(ctx context.Context, id string) (*model.Customer, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid customer id")
	}

	customer, err := uc.repo.GetByID(ctx, uid)
	if err != nil {
		uc.logger.Error("Failed to get customer", zap.Error(err))
		return nil, err
	}
	if customer == nil {
		return nil, errors.New("customer not found")
	}
	return customer, nil
}

func (uc *customerUseCase) ListCustomers(ctx context.Context, merchantID string, page, pageSize int, search string) ([]*model.Customer, int, error) {
	mid, err := uuid.Parse(merchantID)
	if err != nil {
		return nil, 0, errors.New("invalid merchant id")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	customers, total, err := uc.repo.List(ctx, mid, page, pageSize, search)
	if err != nil {
		uc.logger.Error("Failed to list customers", zap.Error(err))
		return nil, 0, err
	}
	return customers, total, nil
}

func (uc *customerUseCase) UpdateCustomer(ctx context.Context, input *model.Customer) (*model.Customer, error) {
	existing, err := uc.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("customer not found")
	}

	// Update fields
	existing.Name = input.Name
	existing.Phone = input.Phone
	existing.Email = input.Email
	existing.Address = input.Address
	existing.UpdatedAt = time.Now()

	if err := uc.repo.Update(ctx, existing); err != nil {
		uc.logger.Error("Failed to update customer", zap.Error(err))
		return nil, err
	}
	return existing, nil
}

func (uc *customerUseCase) DeleteCustomer(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid customer id")
	}
	if err := uc.repo.Delete(ctx, uid); err != nil {
		uc.logger.Error("Failed to delete customer", zap.Error(err))
		return err
	}
	return nil
}

func (uc *customerUseCase) AddLoyaltyPoints(ctx context.Context, customerID string, points int32) (int32, error) {
	uid, err := uuid.Parse(customerID)
	if err != nil {
		return 0, errors.New("invalid customer id")
	}

	if err := uc.repo.AddLoyaltyPoints(ctx, uid, points); err != nil {
		uc.logger.Error("Failed to add loyalty points", zap.Error(err))
		return 0, err
	}

	// Fetch updated customer to get latest points
	updatedCustomer, err := uc.repo.GetByID(ctx, uid)
	if err != nil {
		// Points added but failed to fetch. Return 0 or error?
		// Better to return success but 0 points, or try to estimate?
		// Or strictly return error.
		return 0, err
	}
	if updatedCustomer == nil {
		return 0, errors.New("customer not found after update")
	}

	return updatedCustomer.LoyaltyPoints, nil
}
