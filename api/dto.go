// Package api provides HTTP handler types for the discount service.
package api

import (
	"fmt"

	"discount-service/models"

	decimal "github.com/shopspring/decimal"
)

// ─── Request / Response DTOs ──────────────────────────────────────────────────

// CartDiscountRequest is the JSON body for POST /api/v1/cart/discounts.
type CartDiscountRequest struct {
	CartItems   []CartItemDTO       `json:"cart_items"`
	Customer    CustomerDTO         `json:"customer"`
	PaymentInfo *PaymentInfoDTO     `json:"payment_info,omitempty"`
}

// ValidateCodeRequest is the JSON body for POST /api/v1/discount/validate.
type ValidateCodeRequest struct {
	Code      string         `json:"code"`
	CartItems []CartItemDTO  `json:"cart_items"`
	Customer  CustomerDTO    `json:"customer"`
}

// CartDiscountResponse is the JSON response for the discount calculation endpoint.
type CartDiscountResponse struct {
	OriginalPrice    string            `json:"original_price"`
	FinalPrice       string            `json:"final_price"`
	AppliedDiscounts map[string]string `json:"applied_discounts"`
	Message          string            `json:"message"`
}

// ValidateCodeResponse is the JSON response for the validate endpoint.
type ValidateCodeResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse is used for all error HTTP responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ─── DTO definitions ──────────────────────────────────────────────────────────

type CartItemDTO struct {
	Product  ProductDTO `json:"product"`
	Quantity int        `json:"quantity"`
	Size     string     `json:"size"`
}

type ProductDTO struct {
	ID           string `json:"id"`
	Brand        string `json:"brand"`
	BrandTier    string `json:"brand_tier"` // "premium" | "regular" | "budget"
	Category     string `json:"category"`
	BasePrice    string `json:"base_price"` // string to avoid float precision issues
}

type CustomerDTO struct {
	ID   string `json:"id"`
	Tier string `json:"tier"`
}

type PaymentInfoDTO struct {
	// Method is "CARD" | "UPI" | "VOUCHER:<CODE>" (to apply a voucher at checkout)
	Method   string  `json:"method"`
	BankName *string `json:"bank_name,omitempty"`
	CardType *string `json:"card_type,omitempty"`
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

// ToModelCartItems converts DTO cart items to model cart items.
// Returns an error string (non-empty) if any base_price string is invalid.
func ToModelCartItems(dtos []CartItemDTO) ([]models.CartItem, string) {
	items := make([]models.CartItem, 0, len(dtos))
	for i, dto := range dtos {
		price, err := decimal.NewFromString(dto.Product.BasePrice)
		if err != nil {
			return nil, fmt.Sprintf("cart_items[%d].product.base_price is not a valid decimal: %v", i, err)
		}
		items = append(items, models.CartItem{
			Product: models.Product{
				ID:           dto.Product.ID,
				Brand:        dto.Product.Brand,
				BrandTier:    models.BrandTier(dto.Product.BrandTier),
				Category:     dto.Product.Category,
				BasePrice:    price,
				CurrentPrice: price,
			},
			Quantity: dto.Quantity,
			Size:     dto.Size,
		})
	}
	return items, ""
}

// ToModelCustomer converts a CustomerDTO to models.CustomerProfile.
func ToModelCustomer(dto CustomerDTO) models.CustomerProfile {
	return models.CustomerProfile{ID: dto.ID, Tier: dto.Tier}
}

// ToModelPaymentInfo converts a PaymentInfoDTO to *models.PaymentInfo.
func ToModelPaymentInfo(dto *PaymentInfoDTO) *models.PaymentInfo {
	if dto == nil {
		return nil
	}
	return &models.PaymentInfo{
		Method:   dto.Method,
		BankName: dto.BankName,
		CardType: dto.CardType,
	}
}

// ToDiscountResponse converts a models.DiscountedPrice to the API response DTO.
func ToDiscountResponse(d *models.DiscountedPrice) CartDiscountResponse {
	applied := make(map[string]string, len(d.AppliedDiscounts))
	for k, v := range d.AppliedDiscounts {
		applied[k] = v.StringFixed(2)
	}
	return CartDiscountResponse{
		OriginalPrice:    d.OriginalPrice.StringFixed(2),
		FinalPrice:       d.FinalPrice.StringFixed(2),
		AppliedDiscounts: applied,
		Message:          d.Message,
	}
}
