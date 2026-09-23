package repositories

import (
	"strings"
	"testing"
)

// TestMaterialFilter_AvailableOnly_AddsSQLCondition verifies that setting
// AvailableOnly=true appends an available_quantity > $N clause to the WHERE.
func TestMaterialFilter_AvailableOnly_AddsSQLCondition(t *testing.T) {
	where, extraArgs := buildMaterialListWhere(MaterialFilter{AvailableOnly: true})
	assertContains(t, "AvailableOnly=true", where, "available_quantity > $")
	if len(extraArgs) != 1 {
		t.Errorf("expected 1 extra arg, got %d", len(extraArgs))
	}
	if extraArgs[0] != 0 {
		t.Errorf("available_quantity arg should be 0, got %v", extraArgs[0])
	}
}

// TestMaterialFilter_AvailableOnly_False_NoCondition verifies that AvailableOnly=false
// does not add an available_quantity clause.
func TestMaterialFilter_AvailableOnly_False_NoCondition(t *testing.T) {
	where, _ := buildMaterialListWhere(MaterialFilter{AvailableOnly: false})
	if strings.Contains(where, "available_quantity") {
		t.Errorf("AvailableOnly=false must not add available_quantity condition: %s", where)
	}
}

// TestMaterialFilter_AvailableOnly_CombinedWithActiveOnly verifies that both
// ActiveOnly and AvailableOnly conditions appear together.
func TestMaterialFilter_AvailableOnly_CombinedWithActiveOnly(t *testing.T) {
	where, extraArgs := buildMaterialListWhere(MaterialFilter{ActiveOnly: true, AvailableOnly: true})
	assertContains(t, "combined", where, "active = $")
	assertContains(t, "combined", where, "available_quantity > $")
	if len(extraArgs) != 2 {
		t.Errorf("expected 2 extra args (active, available_quantity), got %d", len(extraArgs))
	}
}

// TestMaterialFilter_AvailableOnly_CombinedWithSearch verifies that AvailableOnly
// and Search can be combined without parameter index collision.
func TestMaterialFilter_AvailableOnly_CombinedWithSearch(t *testing.T) {
	where, extraArgs := buildMaterialListWhere(MaterialFilter{AvailableOnly: true, Search: "beton"})
	assertContains(t, "combined", where, "available_quantity > $")
	assertContains(t, "combined", where, "ILIKE $")
	if len(extraArgs) != 2 {
		t.Errorf("expected 2 extra args, got %d", len(extraArgs))
	}
}

// TestMaterialFilter_AvailableOnly_ParameterIndexNotColliding verifies that
// ActiveOnly, AvailableOnly, and Search use distinct parameter indices.
func TestMaterialFilter_AvailableOnly_ParameterIndexNotColliding(t *testing.T) {
	where, extraArgs := buildMaterialListWhere(MaterialFilter{
		ActiveOnly:    true,
		AvailableOnly: true,
		Search:        "beton",
	})
	// Should have 3 extra args: true (active), 0 (available_quantity), "%beton%" (search)
	if len(extraArgs) != 3 {
		t.Errorf("expected 3 extra args, got %d: %v", len(extraArgs), extraArgs)
	}
	// Check that active=$3, available_quantity=$4, ILIKE $5 — no repeated index.
	assertContains(t, "index-check", where, "active = $3")
	assertContains(t, "index-check", where, "available_quantity > $4")
	assertContains(t, "index-check", where, "ILIKE $5")
}

// TestMaterialFilter_NoFilters_OnlyProjectCompany verifies the base case where
// no optional filter is applied.
func TestMaterialFilter_NoFilters_OnlyProjectCompany(t *testing.T) {
	where, extraArgs := buildMaterialListWhere(MaterialFilter{})
	assertContains(t, "base", where, "project_id = $1")
	assertContains(t, "base", where, "company_id = $2")
	if len(extraArgs) != 0 {
		t.Errorf("expected no extra args for empty filter, got %d", len(extraArgs))
	}
}

// TestMaterialFilter_DailyReportFormDefault_ZeroStockNotFiltered verifies that
// the default query used by the daily-report material picker (ActiveOnly=true,
// AvailableOnly=false) returns zero-stock materials. A material with
// available_quantity=0 but active=true must appear so that demontaža can select
// it and montaža can show it as disabled. Only explicit deactivation (active=false)
// should hide a material.
func TestMaterialFilter_DailyReportFormDefault_ZeroStockNotFiltered(t *testing.T) {
	// Mirrors: activeOnly := c.Query("active") != "false"  (true by default)
	//          availableOnly := c.Query("available_only") == "true"  (false by default)
	where, extraArgs := buildMaterialListWhere(MaterialFilter{ActiveOnly: true, AvailableOnly: false})
	// active filter is present
	assertContains(t, "default", where, "active = $")
	// no quantity filter — zero-stock active materials must be included
	if strings.Contains(where, "available_quantity") {
		t.Errorf("default DailyReportForm query must not filter by available_quantity; got: %s", where)
	}
	// exactly one extra arg: active=true
	if len(extraArgs) != 1 {
		t.Errorf("expected 1 extra arg (active), got %d", len(extraArgs))
	}
}
