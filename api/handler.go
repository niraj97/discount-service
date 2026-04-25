package api

import (
	"encoding/json"
	"net/http"
	"strings"

	service "discount-service/service"
)

// Handler holds the HTTP handlers for the discount service.
type Handler struct {
	svc service.DiscountService
}

// NewHandler constructs a Handler.
func NewHandler(svc service.DiscountService) *Handler {
	return &Handler{svc: svc}
}

// ─── POST /api/v1/cart/discounts ──────────────────────────────────────────────

// CalculateCartDiscounts accepts a cart and returns a fully-priced discount breakdown.
//
// Request body:
//
//	{
//	  "cart_items": [...],
//	  "customer": { "id": "c1", "tier": "regular" },
//	  "payment_info": { "method": "CARD", "bank_name": "ICICI", "card_type": "CREDIT" },
//	  "voucher_code": "SAVE10"
//	}
func (h *Handler) CalculateCartDiscounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CartDiscountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if len(req.CartItems) == 0 {
		writeError(w, http.StatusBadRequest, "cart_items must not be empty")
		return
	}

	cartItems, errMsg := ToModelCartItems(req.CartItems)
	if errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg)
		return
	}

	customer := ToModelCustomer(req.Customer)
	paymentInfo := ToModelPaymentInfo(req.PaymentInfo)

	result, err := h.svc.CalculateCartDiscounts(r.Context(), cartItems, customer, req.VoucherCode, paymentInfo)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ToDiscountResponse(result))
}

// ─── POST /api/v1/discount/validate ──────────────────────────────────────────

// ValidateDiscountCode checks whether a coupon code is applicable to the given cart.
//
// Request body:
//
//	{
//	  "code": "SAVE10",
//	  "cart_items": [...],
//	  "customer": { "id": "c1", "tier": "regular" }
//	}
func (h *Handler) ValidateDiscountCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ValidateCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	cartItems, errMsg := ToModelCartItems(req.CartItems)
	if errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg)
		return
	}

	customer := ToModelCustomer(req.Customer)

	valid, err := h.svc.ValidateDiscountCode(r.Context(), req.Code, cartItems, customer)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, ValidateCodeResponse{Valid: valid})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
