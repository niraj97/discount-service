// Package testdata provides pre-built discount configuration and cart fixtures
// for use in main.go examples and unit tests.
package testdata

import (
	"discount-service/models"
	"discount-service/repository"

	decimal "github.com/shopspring/decimal"
)

// ─── Discount Configuration ───────────────────────────────────────────────────

// BrandDiscounts returns all configured brand-level discounts.
func BrandDiscounts() []repository.BrandDiscount {
	return []repository.BrandDiscount{
		{Brand: "PUMA", Percentage: decimal.NewFromInt(40)}, // 40% brand discount on PUMA
	}
}

// CategoryDiscounts returns all configured category-level discounts.
func CategoryDiscounts() []repository.CategoryDiscount {
	return []repository.CategoryDiscount{
		{Category: "T-Shirts", Percentage: decimal.NewFromInt(10)}, // 10% on all T-Shirts
	}
}

// Vouchers returns available coupon/voucher codes.
func Vouchers() []repository.VoucherCode {
	return []repository.VoucherCode{
		{
			Code:                 "SAVE10",
			Percentage:           decimal.NewFromInt(10),
			MinCartValue:         decimal.NewFromInt(500),
			ExcludedBrands:       []string{},
			ExcludedCategories:   []string{},
			RequiredCustomerTier: "",
		},
		{
			Code:                 "PREMIUM20",
			Percentage:           decimal.NewFromInt(20),
			MinCartValue:         decimal.NewFromInt(1000),
			ExcludedBrands:       []string{},
			ExcludedCategories:   []string{},
			RequiredCustomerTier: "premium", // only for premium customers
		},
		{
			Code:                 "NOTSHOES",
			Percentage:           decimal.NewFromInt(15),
			MinCartValue:         decimal.Zero,
			ExcludedBrands:       []string{},
			ExcludedCategories:   []string{"Shoes"},
			RequiredCustomerTier: "",
		},
		{
			Code:                 "SUPER69",
			Percentage:           decimal.NewFromInt(69),
			MinCartValue:         decimal.Zero,
			ExcludedBrands:       []string{},
			ExcludedCategories:   []string{},
			RequiredCustomerTier: "",
		},
	}
}

// BankOffers returns all configured bank instant-discount offers.
func BankOffers() []repository.BankOffer {
	return []repository.BankOffer{
		{BankName: "ICICI", CardType: "", Percentage: decimal.NewFromInt(10)}, // 10% on any ICICI card
		{BankName: "HDFC", CardType: "CREDIT", Percentage: decimal.NewFromInt(5)},
	}
}

// ─── Cart & Customer Fixtures ─────────────────────────────────────────────────

// str is a helper to get a *string from a literal.
func str(s string) *string { return &s }

// PumaTShirtCart returns a single-item cart with one PUMA T-shirt priced at ₹2000.
func PumaTShirtCart() []models.CartItem {
	return []models.CartItem{
		{
			Product: models.Product{
				ID:           "prod-001",
				Brand:        "PUMA",
				BrandTier:    models.BrandTierPremium,
				Category:     "T-Shirts",
				BasePrice:    decimal.NewFromInt(2000),
				CurrentPrice: decimal.NewFromInt(2000),
			},
			Quantity: 1,
			Size:     "M",
		},
	}
}

// MultiItemCart returns a cart with a PUMA T-shirt and an Adidas sneaker.
func MultiItemCart() []models.CartItem {
	return []models.CartItem{
		{
			Product: models.Product{
				ID:           "prod-001",
				Brand:        "PUMA",
				BrandTier:    models.BrandTierPremium,
				Category:     "T-Shirts",
				BasePrice:    decimal.NewFromInt(2000),
				CurrentPrice: decimal.NewFromInt(2000),
			},
			Quantity: 1,
			Size:     "M",
		},
		{
			Product: models.Product{
				ID:           "prod-002",
				Brand:        "Adidas",
				BrandTier:    models.BrandTierPremium,
				Category:     "Shoes",
				BasePrice:    decimal.NewFromInt(3000),
				CurrentPrice: decimal.NewFromInt(3000),
			},
			Quantity: 1,
			Size:     "42",
		},
	}
}

// DefaultCustomer returns a generic regular-tier customer.
func DefaultCustomer() models.CustomerProfile {
	return models.CustomerProfile{ID: "cust-001", Tier: "regular"}
}

// PremiumCustomer returns a premium-tier customer.
func PremiumCustomer() models.CustomerProfile {
	return models.CustomerProfile{ID: "cust-002", Tier: "premium"}
}

// ICICIPayment returns a PaymentInfo for an ICICI credit card.
func ICICIPayment() *models.PaymentInfo {
	return &models.PaymentInfo{
		Method:   "CARD",
		BankName: str("ICICI"),
		CardType: str("CREDIT"),
	}
}



// HDFCCreditPayment returns a PaymentInfo for an HDFC credit card.
func HDFCCreditPayment() *models.PaymentInfo {
	return &models.PaymentInfo{
		Method:   "CARD",
		BankName: str("HDFC"),
		CardType: str("CREDIT"),
	}
}

// UPIPayment returns a PaymentInfo for a UPI transaction (no bank offers apply).
func UPIPayment() *models.PaymentInfo {
	return &models.PaymentInfo{Method: "UPI"}
}
