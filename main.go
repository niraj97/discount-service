package main

import (
	"fmt"
	"log"
	"net/http"

	"discount-service/api"
	"discount-service/repository"
	service "discount-service/service"
	"discount-service/testdata"
)

func main() {
	// ── Wire up dependencies ──────────────────────────────────────────────────
	repo := repository.NewInMemoryDiscountRepository(
		testdata.BrandDiscounts(),
		testdata.CategoryDiscounts(),
		testdata.Vouchers(),
		testdata.BankOffers(),
	)
	svc := service.NewDiscountService(repo)
	h := api.NewHandler(svc)

	// ── Register routes ───────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/cart/discounts", h.CalculateCartDiscounts)
	mux.HandleFunc("/api/v1/discount/validate", h.ValidateDiscountCode)

	// ── Start server ──────────────────────────────────────────────────────────
	addr := ":8080"
	fmt.Printf("Discount service listening on http://localhost%s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  POST http://localhost:8080/api/v1/cart/discounts")
	fmt.Println("  POST http://localhost:8080/api/v1/discount/validate")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
