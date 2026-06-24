package purchase

import (
	"testing"
)

func TestGeneratePONumber(t *testing.T) {
	po := GeneratePONumber(2026, 1)
	expected := "PO-2026-0001"
	if po != expected {
		t.Errorf("expected %s, got %s", expected, po)
	}

	po2 := GeneratePONumber(2026, 100)
	expected2 := "PO-2026-0100"
	if po2 != expected2 {
		t.Errorf("expected %s, got %s", expected2, po2)
	}
}

func TestPOLineItemCalculate(t *testing.T) {
	item := &POLineItem{
		Quantity:  5,
		UnitPrice: 10.50,
	}
	item.CalculateLineTotal()
	if item.TotalPrice != 52.50 {
		t.Errorf("expected 52.50, got %.2f", item.TotalPrice)
	}
}

func TestPurchaseOrderCalculateTotals(t *testing.T) {
	po := &PurchaseOrder{
		Tax: 10.00,
		LineItems: []POLineItem{
			{Quantity: 2, UnitPrice: 100},
			{Quantity: 3, UnitPrice: 50},
		},
	}
	po.CalculateTotals()
	if po.Subtotal != 350.00 {
		t.Errorf("expected subtotal 350.00, got %.2f", po.Subtotal)
	}
	if po.Total != 360.00 {
		t.Errorf("expected total 360.00, got %.2f", po.Total)
	}
}

func TestCheckLowStock(t *testing.T) {
	config := DefaultAutoPOConfig()

	// Stock below 50% of min → should trigger
	trigger := CheckLowStock(5, 20, config)
	if trigger == nil {
		t.Fatal("expected trigger for stock <= 50% of min")
	}
	if trigger.CurrentStock != 5 {
		t.Errorf("expected current stock 5, got %d", trigger.CurrentStock)
	}

	// Stock above threshold → should NOT trigger
	trigger = CheckLowStock(15, 20, config)
	if trigger != nil {
		t.Error("expected NO trigger for stock above threshold")
	}

	// Min stock = 0 → should NOT trigger
	trigger = CheckLowStock(0, 0, config)
	if trigger != nil {
		t.Error("expected NO trigger for zero min stock")
	}
}

func TestDefaultAutoPOConfig(t *testing.T) {
	cfg := DefaultAutoPOConfig()
	if cfg.MinStockThreshold != 50 {
		t.Errorf("expected threshold 50, got %d", cfg.MinStockThreshold)
	}
	if cfg.DefaultQuantity != 10 {
		t.Errorf("expected default qty 10, got %d", cfg.DefaultQuantity)
	}
}

func TestPOStatusValues(t *testing.T) {
	statuses := []POStatus{PODraft, POSent, POApproved, POReceived, POClosed, POCancelled}
	if len(statuses) != 6 {
		t.Errorf("expected 6 statuses, got %d", len(statuses))
	}
}

func TestGoodsReceipt(t *testing.T) {
	gr := GoodsReceipt{
		ReceivedBy: "tech-001",
		Items: []GoodsReceiptItem{
			{PartID: "part-001", ExpectedQty: 10, ReceivedQty: 9, DamagedQty: 1, Accepted: true},
		},
		Notes: "Minor damage on 1 unit",
	}
	if len(gr.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(gr.Items))
	}
	if gr.ReceivedBy != "tech-001" {
		t.Errorf("expected tech-001, got %s", gr.ReceivedBy)
	}
}

func TestPOSummary(t *testing.T) {
	po := &PurchaseOrder{
		PONumber:   "PO-2026-0001",
		Status:     PODraft,
		VendorName: "TechSupply Co",
		Total:      1500.00,
		LineItems:  []POLineItem{{}, {}},
	}
	summary := po.Summary()
	expected := "PO-2026-0001 [DRAFT] TechSupply Co: $1500.00 (2 items)"
	if summary != expected {
		t.Errorf("expected '%s', got '%s'", expected, summary)
	}
}
