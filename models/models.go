package models

import (
	decimal "github.com/shopspring/decimal"
)

// BrandTier represents the brand tier.
type BrandTier string

// Discount Service constants
const (
	BrandTierPremium BrandTier = "premium"
	BrandTierRegular BrandTier = "regular"
	BrandTierBudget  BrandTier = "budget"
)

// CustomerTier represents the customer tier.
type CustomerTier string

// Discount Service constants
const (
	CustomerTierPremium CustomerTier = "premium"
	CustomerTierRegular CustomerTier = "regular"
	CustomerTierNone    CustomerTier = ""
)

// Product represents a product in the cart.
type Product struct {
	ID           string          `json:"id"`
	Brand        string          `json:"brand"`
	BrandTier    BrandTier       `json:"brand_tier"`
	Category     string          `json:"category"`
	BasePrice    decimal.Decimal `json:"base_price"`
	CurrentPrice decimal.Decimal `json:"current_price"` // After brand/category discount
}

// CartItem represents a single item in the cart.
type CartItem struct {
	Product  Product `json:"product"`
	Quantity int     `json:"quantity"`
	Size     string  `json:"size"`
}

// PaymentInfo represents payment information.
type PaymentInfo struct {
	Method   string  `json:"method"`    // CARD, UPI, etc
	BankName *string `json:"bank_name"` // Optional
	CardType *string `json:"card_type"` // Optional: CREDIT, DEBIT
}

// DiscountedPrice represents the discounted price.
type DiscountedPrice struct {
	OriginalPrice    decimal.Decimal            `json:"original_price"`
	FinalPrice       decimal.Decimal            `json:"final_price"`
	AppliedDiscounts map[string]decimal.Decimal `json:"applied_discounts"` // discount_name -> amount
	Message          string                     `json:"message"`
}

// CustomerProfile represents customer information.
type CustomerProfile struct {
	ID   string       `json:"id"`
	Tier CustomerTier `json:"tier"`
}
