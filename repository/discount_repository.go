// Package repository provides access to discount rules and coupon data.
// It acts as the data layer, keeping business logic decoupled from raw data.
package repository

import (
	"fmt"
	"strings"

	"discount-service/models"

	decimal "github.com/shopspring/decimal"
)

// BrandDiscount describes a percentage discount applied to a specific brand.
type BrandDiscount struct {
	Brand      string
	Percentage decimal.Decimal // e.g., 40 means 40%
}

// CategoryDiscount describes a percentage discount applied to a product category.
type CategoryDiscount struct {
	Category   string
	Percentage decimal.Decimal
}

// VoucherCode describes a discount voucher that a customer can apply at checkout.
type VoucherCode struct {
	Code                 string
	Percentage           decimal.Decimal
	MinCartValue         decimal.Decimal // minimum cart total to apply
	ExcludedBrands       []string        // brands the voucher cannot be used with
	ExcludedCategories   []string        // categories the voucher cannot be used with
	RequiredCustomerTier string          // empty means available to all
}

// IsEligible checks if a voucher can be applied for the given customer and cart items.
func (v *VoucherCode) IsEligible(customer models.CustomerProfile, items []models.CartItem) error {
	if v.RequiredCustomerTier != "" && !strings.EqualFold(v.RequiredCustomerTier, customer.Tier) {
		return fmt.Errorf("discount code %q requires customer tier %q, got %q", v.Code, v.RequiredCustomerTier, customer.Tier)
	}

	for _, item := range items {
		for _, b := range v.ExcludedBrands {
			if strings.EqualFold(item.Product.Brand, b) {
				return fmt.Errorf("discount code %q cannot be applied to brand %q", v.Code, item.Product.Brand)
			}
		}
		for _, c := range v.ExcludedCategories {
			if strings.EqualFold(item.Product.Category, c) {
				return fmt.Errorf("discount code %q cannot be applied to category %q", v.Code, item.Product.Category)
			}
		}
	}
	return nil
}

// BankOffer describes an instant discount for paying with a specific bank card.
type BankOffer struct {
	BankName   string
	CardType   string // "CREDIT", "DEBIT", or "" for any
	Percentage decimal.Decimal
}

// DiscountRepository is the interface for fetching discount configuration data.
type DiscountRepository interface {
	GetBrandDiscounts() []BrandDiscount
	GetCategoryDiscounts() []CategoryDiscount
	GetVoucherByCode(code string) (*VoucherCode, bool)
	GetBankOffers() []BankOffer
}

// inMemoryDiscountRepository is a simple in-memory implementation of DiscountRepository.
type inMemoryDiscountRepository struct {
	brandDiscounts    []BrandDiscount
	categoryDiscounts []CategoryDiscount
	vouchers          map[string]VoucherCode
	bankOffers        []BankOffer
}

// NewInMemoryDiscountRepository returns a DiscountRepository pre-loaded with
// the discount rules defined in testdata.
func NewInMemoryDiscountRepository(
	brandDiscounts []BrandDiscount,
	categoryDiscounts []CategoryDiscount,
	vouchers []VoucherCode,
	bankOffers []BankOffer,
) DiscountRepository {
	voucherMap := make(map[string]VoucherCode, len(vouchers))
	for _, v := range vouchers {
		voucherMap[v.Code] = v
	}
	return &inMemoryDiscountRepository{
		brandDiscounts:    brandDiscounts,
		categoryDiscounts: categoryDiscounts,
		vouchers:          voucherMap,
		bankOffers:        bankOffers,
	}
}

func (r *inMemoryDiscountRepository) GetBrandDiscounts() []BrandDiscount {
	return r.brandDiscounts
}

func (r *inMemoryDiscountRepository) GetCategoryDiscounts() []CategoryDiscount {
	return r.categoryDiscounts
}

func (r *inMemoryDiscountRepository) GetVoucherByCode(code string) (*VoucherCode, bool) {
	v, ok := r.vouchers[code]
	if !ok {
		return nil, false
	}
	return &v, true
}

func (r *inMemoryDiscountRepository) GetBankOffers() []BankOffer {
	return r.bankOffers
}
