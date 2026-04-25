package service

import (
	"context"
	"fmt"
	"strings"

	"discount-service/models"
	"discount-service/repository"
	"discount-service/service/discount"

	decimal "github.com/shopspring/decimal"
)

// discountServiceImpl is the concrete implementation of DiscountService.
// It owns an ordered pipeline of discount handlers that are applied sequentially.
type discountServiceImpl struct {
	repo     repository.DiscountRepository
	pipeline []discount.Handler
}

// NewDiscountService constructs a DiscountService with the standard four-stage
// pipeline: brand → category → voucher → bank offer.
func NewDiscountService(repo repository.DiscountRepository) DiscountService {
	return &discountServiceImpl{
		repo: repo,
		pipeline: []discount.Handler{
			&discount.BrandHandler{Repo: repo},
			&discount.CategoryHandler{Repo: repo},
			&discount.VoucherHandler{Repo: repo},
			&discount.BankOfferHandler{Repo: repo},
		},
	}
}

// CalculateCartDiscounts applies all discount stages in order and returns a
// fully-populated DiscountedPrice with an itemised breakdown.
func (s *discountServiceImpl) CalculateCartDiscounts(
	ctx context.Context,
	cartItems []models.CartItem,
	customer models.CustomerProfile,
	voucherCode string,
	paymentInfo *models.PaymentInfo,
) (*models.DiscountedPrice, error) {
	if len(cartItems) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Compute gross original price and seed per-item running prices.
	// itemPrices[i] starts at base_price * quantity and is reduced in-place
	// by each handler stage: brand → category → voucher → bank.
	itemPrices := make([]decimal.Decimal, len(cartItems))
	originalPrice := decimal.Zero
	for i, item := range cartItems {
		lineTotal := item.Product.BasePrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		itemPrices[i] = lineTotal
		originalPrice = originalPrice.Add(lineTotal)
	}

	appliedDiscounts := make(map[string]decimal.Decimal)

	// Run each handler in strict pipeline order.
	// Handlers mutate itemPrices in-place; each stage compounds on the
	// price produced by the previous stage.
	hCtx := &discount.HandlerContext{
		Items:            cartItems,
		ItemPrices:       itemPrices,
		VoucherCode:      voucherCode,
		PaymentInfo:      paymentInfo,
		Customer:         customer,
		AppliedDiscounts: appliedDiscounts,
	}

	for _, handler := range s.pipeline {
		handler.Apply(hCtx)
	}

	// Sum per-item running prices to get the final cart price.
	finalPrice := decimal.Zero
	for _, p := range itemPrices {
		if p.IsPositive() {
			finalPrice = finalPrice.Add(p)
		}
	}

	totalSaved := originalPrice.Sub(finalPrice)
	message := buildMessage(appliedDiscounts, totalSaved)

	return &models.DiscountedPrice{
		OriginalPrice:    originalPrice,
		FinalPrice:       finalPrice,
		AppliedDiscounts: appliedDiscounts,
		Message:          message,
	}, nil
}

// ValidateDiscountCode checks whether a voucher code is usable for the given
// cart and customer without applying any discounts.
func (s *discountServiceImpl) ValidateDiscountCode(
	ctx context.Context,
	code string,
	cartItems []models.CartItem,
	customer models.CustomerProfile,
) (bool, error) {
	if strings.TrimSpace(code) == "" {
		return false, fmt.Errorf("discount code cannot be empty")
	}

	voucher, found := s.repo.GetVoucherByCode(code)
	if !found {
		return false, fmt.Errorf("discount code %q not found", code)
	}

	// Tier restriction check.
	if voucher.RequiredCustomerTier != "" &&
		!strings.EqualFold(voucher.RequiredCustomerTier, customer.Tier) {
		return false, fmt.Errorf(
			"discount code %q requires customer tier %q, got %q",
			code, voucher.RequiredCustomerTier, customer.Tier,
		)
	}

	// Brand exclusion check.
	for _, item := range cartItems {
		for _, excluded := range voucher.ExcludedBrands {
			if strings.EqualFold(item.Product.Brand, excluded) {
				return false, fmt.Errorf(
					"discount code %q cannot be applied to brand %q",
					code, item.Product.Brand,
				)
			}
		}
		for _, excluded := range voucher.ExcludedCategories {
			if strings.EqualFold(item.Product.Category, excluded) {
				return false, fmt.Errorf(
					"discount code %q cannot be applied to category %q",
					code, item.Product.Category,
				)
			}
		}
	}

	return true, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// buildMessage constructs a human-readable summary of applied discounts.
func buildMessage(appliedDiscounts map[string]decimal.Decimal, totalSaved decimal.Decimal) string {
	if len(appliedDiscounts) == 0 {
		return "No discounts applied."
	}
	msg := fmt.Sprintf("You saved ₹%s in total. Breakdown: ", totalSaved.StringFixed(2))
	parts := make([]string, 0, len(appliedDiscounts))
	for name, amt := range appliedDiscounts {
		parts = append(parts, fmt.Sprintf("%s → ₹%s", name, amt.StringFixed(2)))
	}
	return msg + strings.Join(parts, " | ")
}
