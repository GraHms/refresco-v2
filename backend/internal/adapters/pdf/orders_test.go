package pdf

import (
	"bytes"
	"os"
	"testing"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

func TestPaymentOrders(t *testing.T) {
	loc, _ := time.LoadLocation("Africa/Maputo")
	now := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	nib := ports.PaymentOrder{Code: "LV-1042", Status: d.WithdrawalApproved, Amount: 1200000, Beneficiary: "Ricardo Mondlane", MemberName: "Cardo", MemberRole: "Apresentador · Admin",
		MemberPhone: "843120457", AccountName: "Tu Pra Tu Podcast", AccountHandle: "tupratu", AccountStatus: d.AccountActive, DestKind: d.PayoutNIB,
		DestNumber: "000800000412345678101", DestBank: "BCI", RequestedAt: now.Add(-50 * time.Hour), DueAt: now.Add(22 * time.Hour), ApprovedAt: &now}
	wallet := nib
	wallet.Code, wallet.DestKind, wallet.DestNumber, wallet.DestBank, wallet.Beneficiary, wallet.Amount = "LV-1047", d.PayoutMpesa, "849081123", "", "Edson", 600000
	r := Renderer{Loc: loc}
	for name, orders := range map[string][]ports.PaymentOrder{"single": {nib}, "batch": {nib, wallet}} {
		b, err := r.PaymentOrders(orders, "admin@refres.co", now)
		if err != nil {
			t.Fatal(name, err)
		}
		if !bytes.HasPrefix(b, []byte("%PDF-")) || len(b) < 2000 {
			t.Fatalf("%s: not a PDF (%d bytes)", name, len(b))
		}
		if dir := os.Getenv("PDF_OUT"); dir != "" {
			_ = os.WriteFile(dir+"/"+name+".pdf", b, 0o644)
		}
	}
	if _, err := r.PaymentOrders(nil, "x", now); err == nil {
		t.Fatal("empty batch should fail")
	}
}
