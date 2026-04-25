package models

import (
	decimal "github.com/shopspring/decimal"
)

type BrandTier string

const (
	BrandTierPremium BrandTier = "premium"
	BrandTierRegular BrandTier = "regular"
	BrandTierBudget  BrandTier = "budget"
)

type Product struct {
	ID           string          `json:"id"`
	Brand        string          `json:"brand"`
	BrandTier    BrandTier       `json:"brand_tier"`
	Category     string          `json:"category"`
	BasePrice    decimal.Decimal `json:"base_price"`
	CurrentPrice decimal.Decimal `json:"current_price"` // After brand/category discount
}

type CartItem struct {
	Product  Product `json:"product"`
	Quantity int     `json:"quantity"`
	Size     string  `json:"size"`
}

type PaymentInfo struct {
	Method   string  `json:"method"`    // CARD, UPI, etc
	BankName *string `json:"bank_name"` // Optional
	CardType *string `json:"card_type"` // Optional: CREDIT, DEBIT
}

type DiscountedPrice struct {
	OriginalPrice    decimal.Decimal            `json:"original_price"`
	FinalPrice       decimal.Decimal            `json:"final_price"`
	AppliedDiscounts map[string]decimal.Decimal `json:"applied_discounts"` // discount_name -> amount
	Message          string                     `json:"message"`
}

type CustomerProfile struct {
	// Add relevant customer fields
	ID   string `json:"id"`
	Tier string `json:"tier"`
	// Add more fields as needed
}
