package service

import (
	"context"
	"testing"

	"discount-service/repository"
	"discount-service/testdata"

	decimal "github.com/shopspring/decimal"
)

// newTestService builds a DiscountService wired to the standard testdata fixtures.
func newTestService() DiscountService {
	repo := repository.NewInMemoryDiscountRepository(
		testdata.BrandDiscounts(),
		testdata.CategoryDiscounts(),
		testdata.Vouchers(),
		testdata.BankOffers(),
	)
	return NewDiscountService(repo)
}

// mustDec parses a decimal string and fails the test immediately on error.
func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("invalid decimal %q: %v", s, err)
	}
	return d
}

// ─── CalculateCartDiscounts ───────────────────────────────────────────────────

func TestCalculateCartDiscounts(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	t.Run("error on empty cart", func(t *testing.T) {
		_, err := svc.CalculateCartDiscounts(ctx, nil, testdata.DefaultCustomer(), testdata.UPIPayment())
		if err == nil {
			t.Fatal("expected error for empty cart, got nil")
		}
	})

	t.Run("happy path: PUMA tshirt + ICICI card", func(t *testing.T) {
		// ₹2000 base
		// brand  40% of ₹2000 → −₹800  → ₹1200
		// cat    10% of ₹1200 → −₹120  → ₹1080  (cascades on post-brand price)
		// bank   10% of ₹1080 → −₹108  → ₹972   (cascades on post-cat price)
		result, err := svc.CalculateCartDiscounts(
			ctx,
			testdata.PumaTShirtCart(),
			testdata.DefaultCustomer(),
			testdata.ICICIPayment(),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDecimalEqual(t, "OriginalPrice", result.OriginalPrice, mustDec(t, "2000"))
		assertDecimalEqual(t, "FinalPrice", result.FinalPrice, mustDec(t, "972"))

		assertDiscount(t, result.AppliedDiscounts, "Brand Discount (PUMA)", mustDec(t, "800"))
		assertDiscount(t, result.AppliedDiscounts, "Category Discount (T-Shirts)", mustDec(t, "120"))
		assertDiscount(t, result.AppliedDiscounts, "Bank Offer (ICICI)", mustDec(t, "108"))

		if result.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("all four stages: PUMA tshirt + SAVE10 voucher + ICICI", func(t *testing.T) {
		// ₹2000 base
		// brand   40% of ₹2000 → −₹800   → ₹1200
		// cat     10% of ₹1200 → −₹120   → ₹1080
		// SAVE10  10% of ₹1080 → −₹108   → ₹972
		// ICICI   10% of ₹972  → −₹97.20 → ₹874.80  (final)
		result, err := svc.CalculateCartDiscounts(
			ctx,
			testdata.PumaTShirtCart(),
			testdata.DefaultCustomer(),
			testdata.ICICIPaymentWithVoucher("SAVE10"),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimalEqual(t, "FinalPrice", result.FinalPrice, mustDec(t, "874.80"))
		assertDiscount(t, result.AppliedDiscounts, "Voucher (SAVE10)", mustDec(t, "108"))
		assertDiscount(t, result.AppliedDiscounts, "Bank Offer (ICICI)", mustDec(t, "97.20"))
	})

	t.Run("no discounts: Adidas Shoes + UPI", func(t *testing.T) {
		// Adidas has no brand discount; Shoes has no category discount; UPI has no bank offer.
		shoesCart := testdata.MultiItemCart()[1:] // just the Adidas Shoes item
		result, err := svc.CalculateCartDiscounts(
			ctx,
			shoesCart,
			testdata.DefaultCustomer(),
			testdata.UPIPayment(),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimalEqual(t, "FinalPrice", result.FinalPrice, mustDec(t, "3000"))
		if len(result.AppliedDiscounts) != 0 {
			t.Errorf("expected no applied discounts, got %v", result.AppliedDiscounts)
		}
	})

	t.Run("multi-item: PUMA tshirt + Adidas Shoes + ICICI", func(t *testing.T) {
		// PUMA T-shirt ₹2000:
		//   brand 40% of ₹2000 → −₹800  → ₹1200
		//   cat   10% of ₹1200 → −₹120  → ₹1080
		// Adidas Shoes ₹3000: no brand/cat discount → ₹3000
		// Subtotal before bank = ₹1080 + ₹3000 = ₹4080
		// ICICI 10% of each:
		//   PUMA:   10% of ₹1080 → −₹108  → ₹972
		//   Adidas: 10% of ₹3000 → −₹300  → ₹2700
		// Final = ₹972 + ₹2700 = ₹3672
		result, err := svc.CalculateCartDiscounts(
			ctx,
			testdata.MultiItemCart(),
			testdata.DefaultCustomer(),
			testdata.ICICIPayment(),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDecimalEqual(t, "FinalPrice", result.FinalPrice, mustDec(t, "3672"))
	})

	t.Run("premium voucher skipped for regular customer", func(t *testing.T) {
		// PREMIUM20 requires tier="premium"; regular customer → voucher skipped.
		// brand 40% of ₹2000 → −₹800 → ₹1200
		// cat   10% of ₹1200 → −₹120 → ₹1080
		// voucher skipped
		// ICICI 10% of ₹1080 → −₹108 → ₹972 (final)
		result, err := svc.CalculateCartDiscounts(
			ctx,
			testdata.PumaTShirtCart(),
			testdata.DefaultCustomer(),
			testdata.ICICIPaymentWithVoucher("PREMIUM20"),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result.AppliedDiscounts["Voucher (PREMIUM20)"]; ok {
			t.Error("PREMIUM20 voucher must NOT be applied to a regular customer")
		}
		assertDecimalEqual(t, "FinalPrice", result.FinalPrice, mustDec(t, "972"))
	})

	t.Run("premium voucher applied for premium customer", func(t *testing.T) {
		// brand    40% of ₹2000 → −₹800  → ₹1200
		// cat      10% of ₹1200 → −₹120  → ₹1080
		// PREMIUM20 20% of ₹1080 → −₹216 → ₹864
		// ICICI    10% of ₹864  → −₹86.40 → ₹777.60 (final)
		result, err := svc.CalculateCartDiscounts(
			ctx,
			testdata.PumaTShirtCart(),
			testdata.PremiumCustomer(),
			testdata.ICICIPaymentWithVoucher("PREMIUM20"),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDiscount(t, result.AppliedDiscounts, "Voucher (PREMIUM20)", mustDec(t, "216"))
		assertDiscount(t, result.AppliedDiscounts, "Bank Offer (ICICI)", mustDec(t, "86.40"))
		assertDecimalEqual(t, "FinalPrice", result.FinalPrice, mustDec(t, "777.60"))
	})

	t.Run("HDFC credit card offer applied", func(t *testing.T) {
		// brand 40% of ₹2000 → −₹800 → ₹1200
		// cat   10% of ₹1200 → −₹120 → ₹1080
		// HDFC CREDIT 5% of ₹1080 → −₹54 → ₹1026 (final)
		result, err := svc.CalculateCartDiscounts(
			ctx,
			testdata.PumaTShirtCart(),
			testdata.DefaultCustomer(),
			testdata.HDFCCreditPayment(),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertDiscount(t, result.AppliedDiscounts, "Bank Offer (HDFC)", mustDec(t, "54"))
		assertDecimalEqual(t, "FinalPrice", result.FinalPrice, mustDec(t, "1026"))
	})
}

// ─── ValidateDiscountCode ─────────────────────────────────────────────────────

func TestValidateDiscountCode(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	tests := []struct {
		name      string
		code      string
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "valid code SAVE10 for regular customer",
			code:      "SAVE10",
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "empty code",
			code:      "",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "unknown code",
			code:      "FAKE999",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "PREMIUM20 for regular customer",
			code:      "PREMIUM20",
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ok, err := svc.ValidateDiscountCode(ctx, tc.code, testdata.PumaTShirtCart(), testdata.DefaultCustomer())
			if (err != nil) != tc.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tc.wantErr, err)
			}
			if ok != tc.wantValid {
				t.Errorf("wantValid=%v, got ok=%v", tc.wantValid, ok)
			}
		})
	}

	t.Run("PREMIUM20 valid for premium customer", func(t *testing.T) {
		ok, err := svc.ValidateDiscountCode(ctx, "PREMIUM20", testdata.PumaTShirtCart(), testdata.PremiumCustomer())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected PREMIUM20 to be valid for premium customer")
		}
	})

	t.Run("NOTSHOES rejected when cart has shoes", func(t *testing.T) {
		ok, err := svc.ValidateDiscountCode(ctx, "NOTSHOES", testdata.MultiItemCart(), testdata.DefaultCustomer())
		if err == nil {
			t.Fatal("expected error: voucher excludes Shoes category")
		}
		if ok {
			t.Error("expected ok=false for excluded category")
		}
	})
}

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func assertDecimalEqual(t *testing.T, field string, got, want decimal.Decimal) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s: got %s, want %s", field, got.StringFixed(2), want.StringFixed(2))
	}
}

func assertDiscount(t *testing.T, m map[string]decimal.Decimal, key string, want decimal.Decimal) {
	t.Helper()
	got, ok := m[key]
	if !ok {
		t.Errorf("missing discount key %q", key)
		return
	}
	if !got.Equal(want) {
		t.Errorf("discount[%q]: got %s, want %s", key, got.StringFixed(2), want.StringFixed(2))
	}
}
