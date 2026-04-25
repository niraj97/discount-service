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
// HandlerContext bundles the inputs for a single stage in the discount pipeline.
type HandlerContext struct {
	Items            []models.CartItem
	ItemPrices       []decimal.Decimal // running price per cart line (mutated)
	VoucherCode      string
	PaymentInfo      *models.PaymentInfo
	Customer         models.CustomerProfile
	AppliedDiscounts map[string]decimal.Decimal
}

type Handler interface {
	Apply(ctx *HandlerContext)
}

// ─── Brand Discount Handler ───────────────────────────────────────────────────

// BrandHandler applies brand-level percentage discounts.
// It is always the first stage, so itemPrices[i] == item.BasePrice * qty here.
type BrandHandler struct {
	Repo repository.DiscountRepository
}

func (h *BrandHandler) Apply(ctx *HandlerContext) {
	for i, item := range ctx.Items {
		for _, bd := range h.Repo.GetBrandDiscounts() {
			if !strings.EqualFold(item.Product.Brand, bd.Brand) {
				continue
			}
			discountAmt := ctx.ItemPrices[i].Mul(bd.Percentage).Div(decimal.NewFromInt(100))
			key := "Brand Discount (" + item.Product.Brand + ")"
			ctx.AppliedDiscounts[key] = ctx.AppliedDiscounts[key].Add(discountAmt)
			ctx.ItemPrices[i] = ctx.ItemPrices[i].Sub(discountAmt)
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

func (h *CategoryHandler) Apply(ctx *HandlerContext) {
	for i, item := range ctx.Items {
		for _, cd := range h.Repo.GetCategoryDiscounts() {
			if !strings.EqualFold(item.Product.Category, cd.Category) {
				continue
			}
			// ctx.ItemPrices[i] is already reduced by any brand discount.
			discountAmt := ctx.ItemPrices[i].Mul(cd.Percentage).Div(decimal.NewFromInt(100))
			key := "Category Discount (" + item.Product.Category + ")"
			ctx.AppliedDiscounts[key] = ctx.AppliedDiscounts[key].Add(discountAmt)
			ctx.ItemPrices[i] = ctx.ItemPrices[i].Sub(discountAmt)
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

func (h *VoucherHandler) Apply(ctx *HandlerContext) {
	if ctx.VoucherCode == "" {
		return
	}
	voucher, found := h.Repo.GetVoucherByCode(ctx.VoucherCode)
	if !found {
		return
	}

	if err := voucher.IsEligible(ctx.Customer, ctx.Items); err != nil {
		return
	}

	// Apply the voucher % uniformly across each item's running price so that
	// the total deduction equals cartTotal * pct/100.
	for i := range ctx.ItemPrices {
		discountAmt := ctx.ItemPrices[i].Mul(voucher.Percentage).Div(decimal.NewFromInt(100))
		ctx.AppliedDiscounts["Voucher ("+ctx.VoucherCode+")"] =
			ctx.AppliedDiscounts["Voucher ("+ctx.VoucherCode+")"].Add(discountAmt)
		ctx.ItemPrices[i] = ctx.ItemPrices[i].Sub(discountAmt)
	}
}

// ─── Bank Offer Handler ───────────────────────────────────────────────────────

// BankOfferHandler applies an instant bank card discount on the cart total.
// It is the last stage, so itemPrices[i] carries the fully-reduced price from
// all previous stages; the bank % compounds on top of those reductions.
type BankOfferHandler struct {
	Repo repository.DiscountRepository
}

func (h *BankOfferHandler) Apply(ctx *HandlerContext) {
	if ctx.PaymentInfo == nil || ctx.PaymentInfo.BankName == nil {
		return
	}

	for _, offer := range h.Repo.GetBankOffers() {
		bankMatch := strings.EqualFold(*ctx.PaymentInfo.BankName, offer.BankName)
		cardMatch := offer.CardType == "" ||
			(ctx.PaymentInfo.CardType != nil && strings.EqualFold(*ctx.PaymentInfo.CardType, offer.CardType))

		if !bankMatch || !cardMatch {
			continue
		}

		key := "Bank Offer (" + offer.BankName + ")"
		for i := range ctx.ItemPrices {
			discountAmt := ctx.ItemPrices[i].Mul(offer.Percentage).Div(decimal.NewFromInt(100))
			ctx.AppliedDiscounts[key] = ctx.AppliedDiscounts[key].Add(discountAmt)
			ctx.ItemPrices[i] = ctx.ItemPrices[i].Sub(discountAmt)
		}
	}
}
