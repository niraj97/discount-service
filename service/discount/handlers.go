// Package discount contains discount strategy handlers.
// Each handler is responsible for applying one category of discount,
// making the pipeline independently extensible.
package discount

import (
	"discount-service/models"
	"discount-service/repository"
	"strings"

	decimal "github.com/shopspring/decimal"
)

// Handler is a single step in the discount pipeline.
//
// itemPrices is a slice parallel to items — itemPrices[i] holds the running
// price for items[i] after all previous pipeline stages have been applied.
// Each handler reads from itemPrices, applies its discount, updates itemPrices
// in-place, and records what it deducted into appliedDiscounts.
//
// This design ensures every stage operates on the price produced by the
// previous stage rather than recalculating from the original base price.
type Handler interface {
	Apply(
		items []models.CartItem,
		itemPrices []decimal.Decimal, // running price per cart line (mutated)
		voucherCode string,
		paymentInfo *models.PaymentInfo,
		customer models.CustomerProfile,
		appliedDiscounts map[string]decimal.Decimal,
	)
}

// ─── Brand Discount Handler ───────────────────────────────────────────────────

// BrandHandler applies brand-level percentage discounts.
// It is always the first stage, so itemPrices[i] == item.BasePrice * qty here.
type BrandHandler struct {
	Repo repository.DiscountRepository
}

func (h *BrandHandler) Apply(
	items []models.CartItem,
	itemPrices []decimal.Decimal,
	_ string,
	_ *models.PaymentInfo,
	_ models.CustomerProfile,
	appliedDiscounts map[string]decimal.Decimal,
) {
	for i, item := range items {
		for _, bd := range h.Repo.GetBrandDiscounts() {
			if !strings.EqualFold(item.Product.Brand, bd.Brand) {
				continue
			}
			discountAmt := itemPrices[i].Mul(bd.Percentage).Div(decimal.NewFromInt(100))
			key := "Brand Discount (" + item.Product.Brand + ")"
			appliedDiscounts[key] = appliedDiscounts[key].Add(discountAmt)
			itemPrices[i] = itemPrices[i].Sub(discountAmt)
		}
	}
}

// ─── Category Discount Handler ────────────────────────────────────────────────

// CategoryHandler applies category-level percentage discounts.
// It runs after BrandHandler, so itemPrices[i] is already the post-brand price —
// the category % is applied on that reduced value, not the base price.
type CategoryHandler struct {
	Repo repository.DiscountRepository
}

func (h *CategoryHandler) Apply(
	items []models.CartItem,
	itemPrices []decimal.Decimal,
	_ string,
	_ *models.PaymentInfo,
	_ models.CustomerProfile,
	appliedDiscounts map[string]decimal.Decimal,
) {
	for i, item := range items {
		for _, cd := range h.Repo.GetCategoryDiscounts() {
			if !strings.EqualFold(item.Product.Category, cd.Category) {
				continue
			}
			// itemPrices[i] is already reduced by any brand discount.
			discountAmt := itemPrices[i].Mul(cd.Percentage).Div(decimal.NewFromInt(100))
			key := "Category Discount (" + item.Product.Category + ")"
			appliedDiscounts[key] = appliedDiscounts[key].Add(discountAmt)
			itemPrices[i] = itemPrices[i].Sub(discountAmt)
		}
	}
}

// ─── Voucher/Coupon Handler ───────────────────────────────────────────────────

// VoucherHandler applies a voucher/coupon discount on the cart total.
// It runs after brand+category, so the % is applied on the already-reduced
// per-item prices (summed to a cart total).
type VoucherHandler struct {
	Repo repository.DiscountRepository
}

func (h *VoucherHandler) Apply(
	items []models.CartItem,
	itemPrices []decimal.Decimal,
	voucherCode string,
	_ *models.PaymentInfo,
	customer models.CustomerProfile,
	appliedDiscounts map[string]decimal.Decimal,
) {
	if voucherCode == "" {
		return
	}
	voucher, found := h.Repo.GetVoucherByCode(voucherCode)
	if !found {
		return
	}

	// Tier check.
	if voucher.RequiredCustomerTier != "" &&
		!strings.EqualFold(voucher.RequiredCustomerTier, customer.Tier) {
		return
	}

	// Brand / category exclusion check.
	for _, item := range items {
		for _, excluded := range voucher.ExcludedBrands {
			if strings.EqualFold(item.Product.Brand, excluded) {
				return
			}
		}
		for _, excluded := range voucher.ExcludedCategories {
			if strings.EqualFold(item.Product.Category, excluded) {
				return
			}
		}
	}

	// Apply the voucher % uniformly across each item's running price so that
	// the total deduction equals cartTotal * pct/100.
	for i := range itemPrices {
		discountAmt := itemPrices[i].Mul(voucher.Percentage).Div(decimal.NewFromInt(100))
		appliedDiscounts["Voucher ("+voucherCode+")"] =
			appliedDiscounts["Voucher ("+voucherCode+")"].Add(discountAmt)
		itemPrices[i] = itemPrices[i].Sub(discountAmt)
	}
}

// ─── Bank Offer Handler ───────────────────────────────────────────────────────

// BankOfferHandler applies an instant bank card discount on the cart total.
// It is the last stage, so itemPrices[i] carries the fully-reduced price from
// all previous stages; the bank % compounds on top of those reductions.
type BankOfferHandler struct {
	Repo repository.DiscountRepository
}

func (h *BankOfferHandler) Apply(
	_ []models.CartItem,
	itemPrices []decimal.Decimal,
	_ string,
	paymentInfo *models.PaymentInfo,
	_ models.CustomerProfile,
	appliedDiscounts map[string]decimal.Decimal,
) {
	if paymentInfo == nil || paymentInfo.BankName == nil {
		return
	}

	for _, offer := range h.Repo.GetBankOffers() {
		bankMatch := strings.EqualFold(*paymentInfo.BankName, offer.BankName)
		cardMatch := offer.CardType == "" ||
			(paymentInfo.CardType != nil && strings.EqualFold(*paymentInfo.CardType, offer.CardType))

		if !bankMatch || !cardMatch {
			continue
		}

		key := "Bank Offer (" + offer.BankName + ")"
		for i := range itemPrices {
			discountAmt := itemPrices[i].Mul(offer.Percentage).Div(decimal.NewFromInt(100))
			appliedDiscounts[key] = appliedDiscounts[key].Add(discountAmt)
			itemPrices[i] = itemPrices[i].Sub(discountAmt)
		}
	}
}
