# Discount Service

A production-style Go microservice that calculates and validates e-commerce discounts. It applies brand, category, voucher, and bank-card discounts in a strict, cascading pipeline and exposes the results over a clean HTTP JSON API.

---

## Table of Contents

- [Problem Statement](#problem-statement)
- [Approach](#approach)
- [Architecture](#architecture)
  - [Architecture Diagram](#architecture-diagram)
- [Discount Pipeline](#discount-pipeline)
- [Assumptions](#assumptions)
- [How to Run](#how-to-run)
- [API Reference](#api-reference)
- [How to Test](#how-to-test)
- [Project Structure](#project-structure)
- [Future Improvements](#future-improvements)

---

## Problem Statement

E-commerce platforms apply multiple overlapping discount types — brand promotions, category sales, coupon codes, and bank instant-discount offers — and the order in which they are applied directly affects the final price.

The challenge is to design a system that:

1. Applies discounts in a **deterministic, configurable order**.
2. **Cascades** each discount onto the price left by the previous stage (not the original base price), so the stacking is mathematically correct.
3. Supports **independent extensibility** — adding a new discount type must not require modifying existing logic.
4. Provides a clear, itemised breakdown of every discount applied.

---

## Approach

### Cascading Pipeline

Each discount stage receives the price that the **previous stage produced**, not the original base price. This mirrors real-world e-commerce behaviour:

```
Base Price: ₹2000 (PUMA T-shirt)

Stage 1 — Brand (PUMA 40%):   40% of ₹2000 = ₹800  off → ₹1200
Stage 2 — Category (T-Shirts 10%): 10% of ₹1200 = ₹120 off → ₹1080
Stage 3 — Voucher (SAVE10 10%):    10% of ₹1080 = ₹108 off → ₹972
Stage 4 — Bank (ICICI 10%):        10% of ₹972  = ₹97.20 off → ₹874.80
```

### Strategy Pattern (Handler Pipeline)

Every discount type is encapsulated in its own `Handler` struct that implements a single `Apply(*HandlerContext)` method. The service orchestrates them in order. Adding a new discount type is as simple as implementing `Handler` and registering it — no existing code changes.

### Per-Item Price Tracking

Instead of a single running cart total, the pipeline maintains a `[]decimal.Decimal` slice — one entry per cart line. This lets brand and category handlers operate at item granularity (e.g., only PUMA items get the brand discount) while voucher and bank handlers aggregate the slice for cart-level calculations.

### Precision

All monetary values use [`github.com/shopspring/decimal`](https://github.com/shopspring/decimal) — no floating-point rounding errors.

---

## Architecture

```
discount-service/
│
├── main.go                          Entry point — wires deps, starts HTTP server
│
├── models/
│   └── models.go                   Domain types: Product, CartItem, DiscountedPrice, etc.
│
├── repository/
│   └── discount_repository.go      Data layer — discount rules, voucher definitions,
│                                   bank offers. Interface + in-memory implementation.
│
├── service/
│   ├── services.go                 DiscountService interface (contract)
│   ├── discount_service_impl.go    Concrete implementation — runs the pipeline
│   ├── discount_service_test.go    Table-driven unit tests
│   └── discount/
│       └── handlers.go             HandlerContext struct + four Handler implementations
│                                   (Brand, Category, Voucher, Bank)
│
├── api/
│   ├── dto.go                      Request/Response DTOs + model mappers
│   └── handler.go                  HTTP handlers (thin — delegate to service)
│
└── testdata/
    └── fake_data.go                Pre-built carts, customers, payment fixtures,
                                    and discount rule sets for tests and main.go
```

### Architecture Diagram

![Architecture Diagram](architecture-diagram.png)

### Dependency Flow

```
main.go
  └─▶ repository (data)
  └─▶ service (business logic)
        └─▶ service/discount (handlers)
              └─▶ repository (read-only)
  └─▶ api (HTTP layer)
        └─▶ service
```

No layer imports from a layer above it. The HTTP layer knows nothing about discount rules; the service layer knows nothing about HTTP.

---

## Discount Pipeline

Discounts are applied in this **strict, non-negotiable order**:

| Stage | Type | Applied on |
|-------|------|-----------|
| 1 | Brand discount | Base price of matching items |
| 2 | Category discount | Post-brand price of matching items |
| 3 | Voucher / coupon | Post-category cart total |
| 4 | Bank instant offer | Post-voucher cart total |

### Voucher Eligibility Rules

A voucher is silently skipped (other stages still apply) if any of the following conditions are unmet:

- `code` does not exist in the repository
- Cart total (post brand+category) is below `MinCartValue`
- Customer tier does not match `RequiredCustomerTier`
- Cart contains a brand listed in `ExcludedBrands`
- Cart contains a category listed in `ExcludedCategories`

### Available Test Vouchers

| Code | Discount | Restriction |
|------|----------|-------------|
| `SAVE10` | 10% | Min cart ₹500 |
| `PREMIUM20` | 20% | `tier: premium` only, min cart ₹1000 |
| `NOTSHOES` | 15% | Cannot apply to Shoes category |
| `SUPER69` | 69% | None |

### Available Test Bank Offers

| Bank | Card Type | Discount |
|------|-----------|----------|
| ICICI | Any | 10% |
| HDFC | CREDIT only | 5% |

---

## Assumptions

1. **Voucher passed explicitly** — the voucher code is a first-class parameter on `CalculateCartDiscounts`, not embedded in `PaymentInfo`. This makes the interface explicit and testable.
2. **Silent skip on ineligible vouchers** — an ineligible voucher during cart calculation does not return an error; it is silently skipped and the remaining pipeline continues. `ValidateDiscountCode` is the explicit pre-check endpoint.
3. **Per-item discount isolation** — brand and category discounts only apply to items whose brand/category matches. Items with no matching discount rules are passed through unchanged.
4. **Bank offer applies to all items** — the bank discount is a cart-level offer applied uniformly across all items' running prices.
5. **No price floor below zero** — any item price that would go negative after discounts is clamped to zero.
6. **In-memory data store** — discount rules and voucher definitions are loaded from `testdata/fake_data.go` at startup. Swapping to a database requires only a new `DiscountRepository` implementation.
7. **Monetary precision** — all calculations use `shopspring/decimal`; JSON responses represent money as strings (e.g. `"final_price": "874.80"`) to avoid float64 precision loss.
8. **No authentication** — the API assumes an authenticated context is handled upstream (e.g. by an API gateway).

---

## How to Run

### Prerequisites

- [Go](https://go.dev/dl/) 1.21 or newer installed on your system.
- An internet connection to fetch the single dependency (`github.com/shopspring/decimal`).

### Setup and Execution

1. **Clone the repository and navigate to the directory:**
   ```bash
   git clone <your-repo-url>
   cd discount-service
   ```

2. **Download dependencies:**
   ```bash
   go mod download
   ```

3. **Run the server directly:**
   ```bash
   go run main.go
   ```

   *Alternatively, build and run the binary:*
   ```bash
   go build -o discount-service
   ./discount-service
   ```

Expected output:

```
Discount service listening on http://localhost:8080
Endpoints:
  POST http://localhost:8080/api/v1/cart/discounts
  POST http://localhost:8080/api/v1/discount/validate
```

The server runs until you press `Ctrl+C`.

---

## API Reference

### `POST /api/v1/cart/discounts`

Calculates the final price after applying all applicable discounts.

**Request body:**

```json
{
  "cart_items": [
    {
      "product": {
        "id": "prod-001",
        "brand": "PUMA",
        "brand_tier": "premium",
        "category": "T-Shirts",
        "base_price": "2000"
      },
      "quantity": 1,
      "size": "M"
    }
  ],
  "customer": {
    "id": "cust-001",
    "tier": "regular"
  },
  "voucher_code": "SAVE10",
  "payment_info": {
    "method": "CARD",
    "bank_name": "ICICI",
    "card_type": "CREDIT"
  }
}
```

> `voucher_code` is optional. Omit or set to `""` to skip the voucher stage.  
> `payment_info.bank_name` is optional. Omit to skip the bank offer stage.

**Response `200 OK`:**

```json
{
  "original_price": "2000.00",
  "final_price": "874.80",
  "applied_discounts": {
    "Brand Discount (PUMA)": "800.00",
    "Category Discount (T-Shirts)": "120.00",
    "Voucher (SAVE10)": "108.00",
    "Bank Offer (ICICI)": "97.20"
  },
  "message": "You saved ₹1125.20 in total. ..."
}
```

---

### `POST /api/v1/discount/validate`

Checks whether a voucher code is applicable to a given cart and customer **without** applying it.

**Request body:**

```json
{
  "code": "PREMIUM20",
  "cart_items": [...],
  "customer": { "id": "cust-002", "tier": "premium" }
}
```

**Response `200 OK` (valid):**

```json
{ "valid": true }
```

**Response `400 Bad Request` (ineligible):**

```json
{
  "error": "discount code \"PREMIUM20\" requires customer tier \"premium\", got \"regular\""
}
```

**Response `404 Not Found` (unknown code):**

```json
{ "error": "discount code \"XYZ\" not found" }
```

---

### Error responses

All errors return a consistent envelope:

```json
{ "error": "human-readable description" }
```

| HTTP Status | Meaning |
|-------------|---------|
| `400` | Invalid JSON, empty cart, or validation failure |
| `404` | Discount code not found |
| `405` | Wrong HTTP method |
| `422` | Domain error (e.g. empty cart items list) |

---

## How to Test

### Run all tests

```bash
go test ./...
```

### Run with verbose output

```bash
go test ./service/... -v
```

### Run a specific test

```bash
go test ./service/... -run TestCalculateCartDiscounts/happy_path
```

### Test coverage

```bash
go test ./... -cover
```

### What is tested

**`TestCalculateCartDiscounts`** — 8 sub-tests covering:

| Sub-test | Covers |
|----------|--------|
| `error on empty cart` | Input validation |
| `happy path: PUMA tshirt + ICICI card` | Brand + category + bank stacking |
| `all four stages: ... SAVE10 ... ICICI` | Full 4-stage cascade |
| `no discounts: Adidas Shoes + UPI` | No-discount scenario |
| `multi-item: PUMA + Adidas + ICICI` | Per-item discount isolation |
| `premium voucher skipped for regular customer` | Tier restriction enforcement |
| `premium voucher applied for premium customer` | Tier match + full stack |
| `HDFC credit card offer applied` | Bank offer (non-ICICI) |

**`TestValidateDiscountCode`** — 6 sub-tests covering:

| Sub-test | Covers |
|----------|--------|
| `valid code SAVE10` | Happy path |
| `empty code` | Input validation |
| `unknown code` | Not-found error |
| `PREMIUM20 for regular customer` | Tier restriction |
| `PREMIUM20 valid for premium customer` | Tier match |
| `NOTSHOES rejected when cart has shoes` | Category exclusion |

---

## Project Structure

```
discount-service/
├── api/
│   ├── dto.go                   JSON request/response types + model mappers
│   └── handler.go               HTTP handlers
├── models/
│   └── models.go                Core domain types (no business logic)
├── repository/
│   └── discount_repository.go   Discount data interface + in-memory store
├── service/
│   ├── services.go              DiscountService interface
│   ├── discount_service_impl.go Service implementation + pipeline runner
│   ├── discount_service_test.go Unit tests
│   └── discount/
│       └── handlers.go          HandlerContext + four Handler implementations
├── testdata/
│   └── fake_data.go             Fixture data (carts, customers, rules)
├── main.go                      HTTP server entry point
├── go.mod
└── go.sum
```

---

## Future Improvements

| Area | Improvement |
|------|-------------|
| **Persistence** | Replace in-memory repository with a PostgreSQL or DynamoDB-backed store; the `DiscountRepository` interface requires no changes in the service layer |
| **Discount rule management** | Build an admin API to create, update, and deactivate discount rules at runtime without redeployment |
| **Coupon usage tracking** | Enforce per-user and global usage limits on voucher codes with an atomic counter in Redis or a DB |
| **Observability** | Add structured logging (e.g. `slog`), Prometheus metrics per discount stage, and distributed tracing (OpenTelemetry) |
| **Authentication** | Add JWT/OAuth2 middleware to validate customer identity before applying tier-restricted vouchers |
| **Handler unit tests** | Write isolated tests for each `Handler` implementation using a mock `DiscountRepository` |
| **Conflict resolution** | Define a policy for mutually-exclusive promotions (e.g. "brand discount cannot stack with a voucher on the same brand") |
| **Partial cart discounts** | Support item-level coupons that apply only to specific SKUs rather than the whole cart |
| **Configuration** | Load discount rules from a config file or environment variables at startup for 12-factor app compliance |
| **API versioning** | Formalise `v1` path prefix with version negotiation middleware |
