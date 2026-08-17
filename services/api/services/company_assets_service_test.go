package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Mock repository ───────────────────────────────────────────────────────────

type mockAssetRepo struct {
	listFn                         func(ctx context.Context, companyID string, f repositories.CompanyAssetsFilter) ([]repositories.CompanyAssetRow, error)
	getByIDFn                      func(ctx context.Context, companyID, id string) (*repositories.CompanyAssetRow, error)
	createFn                       func(ctx context.Context, companyID string, row repositories.CompanyAssetRow) (string, error)
	updateFn                       func(ctx context.Context, companyID, id string, row repositories.CompanyAssetRow) error
	deactivateFn                   func(ctx context.Context, companyID, id string) error
	listActiveEmployeesFn          func(ctx context.Context, companyID string) ([]repositories.AssetEmployeeRow, error)
	employeeBelongsToCompanyFn     func(ctx context.Context, companyID, employeeID string) (bool, error)
	createLeasingPaymentFn         func(ctx context.Context, row repositories.LeasingPaymentRow) (*repositories.LeasingPaymentRow, error)
	listCompletedLeasingForPeriodFn func(ctx context.Context, companyID string, periodMonth time.Time) (map[string]bool, error)
	getLeasingPaymentsByAssetFn    func(ctx context.Context, companyID, assetID string) ([]repositories.LeasingPaymentRow, error)
}

func (m *mockAssetRepo) List(ctx context.Context, companyID string, f repositories.CompanyAssetsFilter) ([]repositories.CompanyAssetRow, error) {
	if m.listFn != nil {
		return m.listFn(ctx, companyID, f)
	}
	return nil, nil
}
func (m *mockAssetRepo) GetByID(ctx context.Context, companyID, id string) (*repositories.CompanyAssetRow, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, companyID, id)
	}
	return &repositories.CompanyAssetRow{ID: id, CompanyID: companyID, AssetType: "alat", Name: "Test", Status: "active"}, nil
}
func (m *mockAssetRepo) Create(ctx context.Context, companyID string, row repositories.CompanyAssetRow) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, companyID, row)
	}
	return "new-asset-id", nil
}
func (m *mockAssetRepo) Update(ctx context.Context, companyID, id string, row repositories.CompanyAssetRow) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, companyID, id, row)
	}
	return nil
}
func (m *mockAssetRepo) Deactivate(ctx context.Context, companyID, id string) error {
	if m.deactivateFn != nil {
		return m.deactivateFn(ctx, companyID, id)
	}
	return nil
}
func (m *mockAssetRepo) ListActiveEmployees(ctx context.Context, companyID string) ([]repositories.AssetEmployeeRow, error) {
	if m.listActiveEmployeesFn != nil {
		return m.listActiveEmployeesFn(ctx, companyID)
	}
	return nil, nil
}
func (m *mockAssetRepo) EmployeeBelongsToCompany(ctx context.Context, companyID, employeeID string) (bool, error) {
	if m.employeeBelongsToCompanyFn != nil {
		return m.employeeBelongsToCompanyFn(ctx, companyID, employeeID)
	}
	return true, nil
}
func (m *mockAssetRepo) CreateLeasingPayment(ctx context.Context, row repositories.LeasingPaymentRow) (*repositories.LeasingPaymentRow, error) {
	if m.createLeasingPaymentFn != nil {
		return m.createLeasingPaymentFn(ctx, row)
	}
	return &repositories.LeasingPaymentRow{
		ID: "pay-1", CompanyID: row.CompanyID, AssetID: row.AssetID,
		PeriodMonth: row.PeriodMonth, CompletedAt: time.Now(), CompletedBy: row.CompletedBy,
	}, nil
}
func (m *mockAssetRepo) ListCompletedLeasingForPeriod(ctx context.Context, companyID string, periodMonth time.Time) (map[string]bool, error) {
	if m.listCompletedLeasingForPeriodFn != nil {
		return m.listCompletedLeasingForPeriodFn(ctx, companyID, periodMonth)
	}
	return map[string]bool{}, nil
}
func (m *mockAssetRepo) GetLeasingPaymentsByAsset(ctx context.Context, companyID, assetID string) ([]repositories.LeasingPaymentRow, error) {
	if m.getLeasingPaymentsByAssetFn != nil {
		return m.getLeasingPaymentsByAssetFn(ctx, companyID, assetID)
	}
	return nil, nil
}

func newAssetSvc(repo *mockAssetRepo) *CompanyAssetsService {
	return NewCompanyAssetsService(repo)
}

// ── Test helpers ──────────────────────────────────────────────────────────────

func makeWarrantyAsset(daysFromToday int, today time.Time) repositories.CompanyAssetRow {
	expiry := today.AddDate(0, 0, daysFromToday)
	return repositories.CompanyAssetRow{
		ID: "a1", Name: "Čekić", AssetType: "alat", Status: "active",
		WarrantyExpiresAt: &expiry,
	}
}

func makeVehicleAsset(daysFromToday int, today time.Time) repositories.CompanyAssetRow {
	expiry := today.AddDate(0, 0, daysFromToday)
	return repositories.CompanyAssetRow{
		ID: "v1", Name: "Kombi", AssetType: "vozilo", Status: "active",
		RegistrationExpiresAt: &expiry,
	}
}

func makeLeasingVehicle(id, name string) repositories.CompanyAssetRow {
	return repositories.CompanyAssetRow{
		ID: id, Name: name, AssetType: "vozilo", Status: "active",
		IsLeasing: true,
	}
}

// ── Role access control ───────────────────────────────────────────────────────

func TestCompanyAssets_AdminOnly_List(t *testing.T) {
	svc := newAssetSvc(&mockAssetRepo{})
	_, err := svc.List(context.Background(), "company-1", "administracija", dto.CompanyAssetFilter{})
	if err != nil {
		t.Fatalf("administracija should be able to list assets, got: %v", err)
	}
}

func TestCompanyAssets_ForbiddenDirector(t *testing.T) {
	svc := newAssetSvc(&mockAssetRepo{})
	_, err := svc.List(context.Background(), "company-1", "direktor", dto.CompanyAssetFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for direktor, got: %v", err)
	}
}

func TestCompanyAssets_ForbiddenInzenjer(t *testing.T) {
	svc := newAssetSvc(&mockAssetRepo{})
	_, err := svc.List(context.Background(), "company-1", "inzenjer", dto.CompanyAssetFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for inzenjer, got: %v", err)
	}
}

func TestCompanyAssets_ForbiddenPoslovoda(t *testing.T) {
	svc := newAssetSvc(&mockAssetRepo{})
	_, err := svc.List(context.Background(), "company-1", "poslovoda", dto.CompanyAssetFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for poslovoda, got: %v", err)
	}
}

func TestCompanyAssets_ForbiddenRadnik(t *testing.T) {
	svc := newAssetSvc(&mockAssetRepo{})
	_, err := svc.List(context.Background(), "company-1", "radnik", dto.CompanyAssetFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for radnik, got: %v", err)
	}
}

// ── Company isolation ─────────────────────────────────────────────────────────

func TestCompanyAssets_CannotAssignEmployeeFromAnotherCompany(t *testing.T) {
	repo := &mockAssetRepo{
		employeeBelongsToCompanyFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "alat", Name: "Čekić", Status: "active"}, nil
		},
	}
	svc := newAssetSvc(repo)

	empID := "foreign-emp-id"
	_, err := svc.Create(context.Background(), "company-1", "administracija", dto.CreateCompanyAssetRequest{
		AssetType:          "alat",
		Name:               "Čekić",
		AssignedEmployeeID: &empID,
	})
	if err == nil {
		t.Fatal("expected error when assigning employee from another company")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got: %v", err)
	}
}

// ── Vehicle registration date validation ──────────────────────────────────────

func TestCompanyAssets_VehicleExpiryCannotPrecedeRegistration(t *testing.T) {
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "vozilo", Name: "Van", Status: "active"}, nil
		},
	}
	svc := newAssetSvc(repo)

	regDate := "2026-06-01"
	expDate := "2026-05-01"
	_, err := svc.Create(context.Background(), "company-1", "administracija", dto.CreateCompanyAssetRequest{
		AssetType:             "vozilo",
		Name:                  "Van",
		RegistrationDate:      &regDate,
		RegistrationExpiresAt: &expDate,
	})
	if err == nil {
		t.Fatal("expected error when expiry precedes registration date")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got: %v", err)
	}
}

func TestCompanyAssets_VehicleExpiryEqualToRegistrationIsValid(t *testing.T) {
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "vozilo", Name: "Van", Status: "active"}, nil
		},
	}
	svc := newAssetSvc(repo)

	regDate := "2026-06-01"
	expDate := "2026-06-01"
	_, err := svc.Create(context.Background(), "company-1", "administracija", dto.CreateCompanyAssetRequest{
		AssetType:             "vozilo",
		Name:                  "Van",
		RegistrationDate:      &regDate,
		RegistrationExpiresAt: &expDate,
	})
	if err != nil {
		t.Fatalf("same-date registration and expiry should be valid, got: %v", err)
	}
}

// ── Equipment size defaults ───────────────────────────────────────────────────

func TestCompanyAssets_EquipmentSizeDefaultsToM(t *testing.T) {
	var capturedSize *string
	repo := &mockAssetRepo{
		createFn: func(_ context.Context, _ string, row repositories.CompanyAssetRow) (string, error) {
			capturedSize = row.Size
			return "asset-id", nil
		},
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			m := "M"
			return &repositories.CompanyAssetRow{ID: id, AssetType: "oprema", Name: "Jakna", Status: "active", Size: &m}, nil
		},
	}
	svc := newAssetSvc(repo)

	_, err := svc.Create(context.Background(), "company-1", "administracija", dto.CreateCompanyAssetRequest{
		AssetType: "oprema",
		Name:      "Jakna",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSize == nil || *capturedSize != "M" {
		t.Errorf("expected size to default to M, got: %v", capturedSize)
	}
}

func TestCompanyAssets_EquipmentSizeExplicitlySet(t *testing.T) {
	var capturedSize *string
	repo := &mockAssetRepo{
		createFn: func(_ context.Context, _ string, row repositories.CompanyAssetRow) (string, error) {
			capturedSize = row.Size
			return "asset-id", nil
		},
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			xl := "XL"
			return &repositories.CompanyAssetRow{ID: id, AssetType: "oprema", Name: "Hlače", Status: "active", Size: &xl}, nil
		},
	}
	svc := newAssetSvc(repo)

	xl := "XL"
	_, err := svc.Create(context.Background(), "company-1", "administracija", dto.CreateCompanyAssetRequest{
		AssetType: "oprema",
		Name:      "Hlače",
		Size:      &xl,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSize == nil || *capturedSize != "XL" {
		t.Errorf("expected size XL, got: %v", capturedSize)
	}
}

// ── Notification boundary rules ───────────────────────────────────────────────

func TestCompanyAssets_NoWarningAt11Days(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeWarrantyAsset(11, today)}, today, nil)
	if len(notes) != 0 {
		t.Errorf("expected no notification at 11 days, got %d", len(notes))
	}
}

func TestCompanyAssets_WarrantyWarningAt10Days(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeWarrantyAsset(10, today)}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expected 1 notification at 10 days, got %d", len(notes))
	}
	if notes[0].State != dto.NotificationStateWarning {
		t.Errorf("expected state=warning, got: %s", notes[0].State)
	}
	if notes[0].DaysRemaining != 10 {
		t.Errorf("expected DaysRemaining=10, got: %d", notes[0].DaysRemaining)
	}
}

func TestCompanyAssets_WarrantyWarningAt1Day(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeWarrantyAsset(1, today)}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expected 1 notification at 1 day, got %d", len(notes))
	}
	if notes[0].State != dto.NotificationStateWarning {
		t.Errorf("expected state=warning, got: %s", notes[0].State)
	}
}

func TestCompanyAssets_WarrantyUrgentOnExpiryDate(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeWarrantyAsset(0, today)}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expected 1 notification on expiry date, got %d", len(notes))
	}
	if notes[0].State != dto.NotificationStateUrgent {
		t.Errorf("expected state=urgent, got: %s", notes[0].State)
	}
	if notes[0].DaysRemaining != 0 {
		t.Errorf("expected DaysRemaining=0, got: %d", notes[0].DaysRemaining)
	}
}

func TestCompanyAssets_WarrantyExpiredAfterDate(t *testing.T) {
	today := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeWarrantyAsset(-3, today)}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expected 1 notification for expired asset, got %d", len(notes))
	}
	if notes[0].State != dto.NotificationStateExpired {
		t.Errorf("expected state=expired, got: %s", notes[0].State)
	}
	if notes[0].DaysRemaining != -3 {
		t.Errorf("expected DaysRemaining=-3, got: %d", notes[0].DaysRemaining)
	}
}

func TestCompanyAssets_ExpiredWarningRemainsVisible(t *testing.T) {
	today := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	longPast := today.AddDate(0, -6, 0)
	asset := repositories.CompanyAssetRow{
		ID: "a2", Name: "Bušilica", AssetType: "alat", Status: "active",
		WarrantyExpiresAt: &longPast,
	}
	notes := ComputeNotifications([]repositories.CompanyAssetRow{asset}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expired notification should remain visible, got %d notifications", len(notes))
	}
	if notes[0].State != dto.NotificationStateExpired {
		t.Errorf("expected state=expired, got: %s", notes[0].State)
	}
}

// ── Vehicle registration warnings ─────────────────────────────────────────────

func TestCompanyAssets_VehicleRegistrationNoWarningAt11Days(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeVehicleAsset(11, today)}, today, nil)
	if len(notes) != 0 {
		t.Errorf("expected no vehicle notification at 11 days, got %d", len(notes))
	}
}

func TestCompanyAssets_VehicleRegistrationWarningAt10Days(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeVehicleAsset(10, today)}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expected 1 vehicle notification at 10 days, got %d", len(notes))
	}
	if notes[0].Kind != dto.NotificationKindRegistration {
		t.Errorf("expected kind=registration, got: %s", notes[0].Kind)
	}
	if notes[0].State != dto.NotificationStateWarning {
		t.Errorf("expected state=warning, got: %s", notes[0].State)
	}
}

func TestCompanyAssets_VehicleRegistrationUrgentOnExpiryDate(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeVehicleAsset(0, today)}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notes))
	}
	if notes[0].State != dto.NotificationStateUrgent {
		t.Errorf("expected state=urgent, got: %s", notes[0].State)
	}
}

func TestCompanyAssets_VehicleRegistrationExpired(t *testing.T) {
	today := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeVehicleAsset(-30, today)}, today, nil)
	if len(notes) != 1 {
		t.Fatalf("expected 1 expired vehicle notification, got %d", len(notes))
	}
	if notes[0].State != dto.NotificationStateExpired {
		t.Errorf("expected state=expired, got: %s", notes[0].State)
	}
}

// ── Notification messages ─────────────────────────────────────────────────────

func TestCompanyAssets_NotificationMessage_WarrantyWarning(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeWarrantyAsset(7, today)}, today, nil)
	if len(notes) == 0 {
		t.Fatal("expected notification")
	}
	want := "Garancija istječe za 7 dana"
	if notes[0].Message != want {
		t.Errorf("message: got %q, want %q", notes[0].Message, want)
	}
}

func TestCompanyAssets_NotificationMessage_RegistrationExpired(t *testing.T) {
	today := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeVehicleAsset(-3, today)}, today, nil)
	if len(notes) == 0 {
		t.Fatal("expected notification")
	}
	want := "Registracija je istekla prije 3 dana"
	if notes[0].Message != want {
		t.Errorf("message: got %q, want %q", notes[0].Message, want)
	}
}

func TestCompanyAssets_NotificationMessage_Today(t *testing.T) {
	today := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeVehicleAsset(0, today)}, today, nil)
	if len(notes) == 0 {
		t.Fatal("expected notification")
	}
	want := "Registracija istječe danas"
	if notes[0].Message != want {
		t.Errorf("message: got %q, want %q", notes[0].Message, want)
	}
}

// ── Leasing notifications – ComputeNotifications ─────────────────────────────

func TestCompanyAssets_Leasing_NonLeasingVehicleNoNotification(t *testing.T) {
	// Vehicle with is_leasing = false must not produce a leasing notification.
	today := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	asset := repositories.CompanyAssetRow{
		ID: "v1", Name: "Kombi", AssetType: "vozilo", Status: "active",
		IsLeasing: false,
	}
	notes := ComputeNotifications([]repositories.CompanyAssetRow{asset}, today, nil)
	for _, n := range notes {
		if n.Kind == dto.NotificationKindLeasingWarning || n.Kind == dto.NotificationKindLeasingOverdue {
			t.Errorf("expected no leasing notification for non-leasing vehicle, got: %v", n)
		}
	}
}

func TestCompanyAssets_Leasing_WarningOnDay1(t *testing.T) {
	today := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeLeasingVehicle("v1", "VW Caddy")}, today, nil)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) != 1 {
		t.Fatalf("expected 1 leasing notification on day 1, got %d", len(leasingNotes))
	}
	if leasingNotes[0].Kind != dto.NotificationKindLeasingWarning {
		t.Errorf("expected kind=leasing_warning on day 1, got: %s", leasingNotes[0].Kind)
	}
	if leasingNotes[0].State != dto.NotificationStateWarning {
		t.Errorf("expected state=warning on day 1, got: %s", leasingNotes[0].State)
	}
	if leasingNotes[0].PeriodMonth != "2026-09-01" {
		t.Errorf("expected period_month=2026-09-01, got: %s", leasingNotes[0].PeriodMonth)
	}
}

func TestCompanyAssets_Leasing_WarningOnDay15(t *testing.T) {
	today := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeLeasingVehicle("v1", "VW Caddy")}, today, nil)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) != 1 {
		t.Fatalf("expected 1 leasing notification on day 15, got %d", len(leasingNotes))
	}
	if leasingNotes[0].Kind != dto.NotificationKindLeasingWarning {
		t.Errorf("expected kind=leasing_warning on day 15, got: %s", leasingNotes[0].Kind)
	}
	if leasingNotes[0].DaysRemaining != 0 {
		t.Errorf("expected DaysRemaining=0 on day 15 (due date = 15th), got: %d", leasingNotes[0].DaysRemaining)
	}
}

func TestCompanyAssets_Leasing_OverdueOnDay16(t *testing.T) {
	today := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeLeasingVehicle("v1", "VW Caddy")}, today, nil)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) != 1 {
		t.Fatalf("expected 1 leasing notification on day 16, got %d", len(leasingNotes))
	}
	if leasingNotes[0].Kind != dto.NotificationKindLeasingOverdue {
		t.Errorf("expected kind=leasing_overdue on day 16, got: %s", leasingNotes[0].Kind)
	}
	if leasingNotes[0].State != dto.NotificationStateUrgent {
		t.Errorf("expected state=urgent on day 16, got: %s", leasingNotes[0].State)
	}
	// DaysRemaining should be negative (past due date).
	if leasingNotes[0].DaysRemaining >= 0 {
		t.Errorf("expected negative DaysRemaining on day 16, got: %d", leasingNotes[0].DaysRemaining)
	}
}

func TestCompanyAssets_Leasing_OverdueRemainsOnDay30(t *testing.T) {
	today := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeLeasingVehicle("v1", "VW Caddy")}, today, nil)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) != 1 {
		t.Fatalf("leasing notification must remain visible on day 30, got %d", len(leasingNotes))
	}
	if leasingNotes[0].Kind != dto.NotificationKindLeasingOverdue {
		t.Errorf("expected kind=leasing_overdue on day 30, got: %s", leasingNotes[0].Kind)
	}
}

func TestCompanyAssets_Leasing_CompletedMonthHidesNotification(t *testing.T) {
	today := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	asset := makeLeasingVehicle("v1", "VW Caddy")
	// Mark September as completed.
	completed := map[string]bool{"v1": true}
	notes := ComputeNotifications([]repositories.CompanyAssetRow{asset}, today, completed)
	for _, n := range notes {
		if n.Kind == dto.NotificationKindLeasingWarning || n.Kind == dto.NotificationKindLeasingOverdue {
			t.Errorf("completed month should not generate a leasing notification, got: %v", n)
		}
	}
}

func TestCompanyAssets_Leasing_CompletingSeptemberDoesNotCompleteOctober(t *testing.T) {
	// September is completed but October is not.
	todayOctober := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	asset := makeLeasingVehicle("v1", "VW Caddy")
	// completed map is for October's period month (2026-10-01), asset NOT in it.
	completed := map[string]bool{} // October not yet completed
	notes := ComputeNotifications([]repositories.CompanyAssetRow{asset}, todayOctober, completed)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) != 1 {
		t.Fatalf("expected 1 leasing notification for October (September completed separately), got %d", len(leasingNotes))
	}
	if leasingNotes[0].PeriodMonth != "2026-10-01" {
		t.Errorf("expected period_month=2026-10-01 for October notification, got: %s", leasingNotes[0].PeriodMonth)
	}
}

func TestCompanyAssets_Leasing_EndDatePreventsFutureReminders(t *testing.T) {
	// Leasing ended August 31. September 1 should produce no notification.
	today := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	asset := repositories.CompanyAssetRow{
		ID: "v1", Name: "VW Caddy", AssetType: "vozilo", Status: "active",
		IsLeasing: true, LeasingEndDate: &endDate,
	}
	notes := ComputeNotifications([]repositories.CompanyAssetRow{asset}, today, nil)
	for _, n := range notes {
		if n.Kind == dto.NotificationKindLeasingWarning || n.Kind == dto.NotificationKindLeasingOverdue {
			t.Errorf("leasing ended before this month — no notification expected, got: %v", n)
		}
	}
}

func TestCompanyAssets_Leasing_EndDateInSameMonthShowsNotification(t *testing.T) {
	// Leasing ends September 30; on September 5 a reminder should still show.
	today := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	asset := repositories.CompanyAssetRow{
		ID: "v1", Name: "VW Caddy", AssetType: "vozilo", Status: "active",
		IsLeasing: true, LeasingEndDate: &endDate,
	}
	notes := ComputeNotifications([]repositories.CompanyAssetRow{asset}, today, nil)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) != 1 {
		t.Fatalf("expected 1 leasing notification when end date is this month, got %d", len(leasingNotes))
	}
}

func TestCompanyAssets_Leasing_DisabledLeasingPreventReminder(t *testing.T) {
	today := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	asset := repositories.CompanyAssetRow{
		ID: "v1", Name: "Kombi", AssetType: "vozilo", Status: "active",
		IsLeasing: false, // explicitly disabled
	}
	notes := ComputeNotifications([]repositories.CompanyAssetRow{asset}, today, nil)
	for _, n := range notes {
		if n.Kind == dto.NotificationKindLeasingWarning || n.Kind == dto.NotificationKindLeasingOverdue {
			t.Errorf("expected no leasing notification when is_leasing=false, got: %v", n)
		}
	}
}

func TestCompanyAssets_Leasing_MessageWarning(t *testing.T) {
	today := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeLeasingVehicle("v1", "VW Caddy")}, today, nil)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) == 0 {
		t.Fatal("expected leasing notification")
	}
	want := "Leasing za rujan 2026. potrebno je riješiti do 15.09.2026."
	if leasingNotes[0].Message != want {
		t.Errorf("message: got %q, want %q", leasingNotes[0].Message, want)
	}
}

func TestCompanyAssets_Leasing_MessageOverdue(t *testing.T) {
	today := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	notes := ComputeNotifications([]repositories.CompanyAssetRow{makeLeasingVehicle("v1", "VW Caddy")}, today, nil)
	leasingNotes := filterLeasing(notes)
	if len(leasingNotes) == 0 {
		t.Fatal("expected leasing notification")
	}
	want := "Leasing za rujan 2026. nije riješen. Rok je bio 15.09.2026."
	if leasingNotes[0].Message != want {
		t.Errorf("message: got %q, want %q", leasingNotes[0].Message, want)
	}
}

// ── Leasing MarkLeasingPayment – service method ───────────────────────────────

func TestCompanyAssets_MarkLeasing_NonAdminForbidden(t *testing.T) {
	svc := newAssetSvc(&mockAssetRepo{})
	_, err := svc.MarkLeasingPayment(context.Background(), "c1", "direktor", "u1", "v1",
		dto.LeasingPaymentRequest{PeriodMonth: "2026-09-01"})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for non-admin, got: %v", err)
	}
}

func TestCompanyAssets_MarkLeasing_NonVehicleRejected(t *testing.T) {
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "alat", Name: "Čekić", Status: "active"}, nil
		},
	}
	svc := newAssetSvc(repo)
	_, err := svc.MarkLeasingPayment(context.Background(), "c1", "administracija", "u1", "a1",
		dto.LeasingPaymentRequest{PeriodMonth: "2026-09-01"})
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError for non-vehicle asset, got: %v", err)
	}
}

func TestCompanyAssets_MarkLeasing_NonLeasingVehicleRejected(t *testing.T) {
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "vozilo", Name: "Kombi", Status: "active", IsLeasing: false}, nil
		},
	}
	svc := newAssetSvc(repo)
	_, err := svc.MarkLeasingPayment(context.Background(), "c1", "administracija", "u1", "v1",
		dto.LeasingPaymentRequest{PeriodMonth: "2026-09-01"})
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError for non-leasing vehicle, got: %v", err)
	}
}

func TestCompanyAssets_MarkLeasing_CannotMarkAnotherCompanysVehicle(t *testing.T) {
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, companyID, id string) (*repositories.CompanyAssetRow, error) {
			if companyID != "company-A" {
				return nil, ErrNotFound // repo enforces company isolation
			}
			return &repositories.CompanyAssetRow{ID: id, AssetType: "vozilo", IsLeasing: true, Status: "active"}, nil
		},
	}
	svc := newAssetSvc(repo)
	// company-B trying to mark company-A's vehicle
	_, err := svc.MarkLeasingPayment(context.Background(), "company-B", "administracija", "u1", "v1",
		dto.LeasingPaymentRequest{PeriodMonth: "2026-09-01"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound when accessing another company's vehicle, got: %v", err)
	}
}

func TestCompanyAssets_MarkLeasing_DuplicateMonthRejected(t *testing.T) {
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "vozilo", IsLeasing: true, Status: "active"}, nil
		},
		createLeasingPaymentFn: func(_ context.Context, row repositories.LeasingPaymentRow) (*repositories.LeasingPaymentRow, error) {
			return nil, repositories.ErrDuplicateLeasingPayment
		},
	}
	svc := newAssetSvc(repo)
	_, err := svc.MarkLeasingPayment(context.Background(), "c1", "administracija", "u1", "v1",
		dto.LeasingPaymentRequest{PeriodMonth: "2026-09-01"})
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError for duplicate month, got: %v", err)
	}
}

func TestCompanyAssets_MarkLeasing_AfterEndDateRejected(t *testing.T) {
	endDate := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{
				ID: id, AssetType: "vozilo", IsLeasing: true, Status: "active",
				LeasingEndDate: &endDate,
			}, nil
		},
	}
	svc := newAssetSvc(repo)
	// Trying to complete September which is after the August 31 end date.
	_, err := svc.MarkLeasingPayment(context.Background(), "c1", "administracija", "u1", "v1",
		dto.LeasingPaymentRequest{PeriodMonth: "2026-09-01"})
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError when marking month after leasing end date, got: %v", err)
	}
}

func TestCompanyAssets_MarkLeasing_Success(t *testing.T) {
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "vozilo", IsLeasing: true, Status: "active"}, nil
		},
	}
	svc := newAssetSvc(repo)
	payment, err := svc.MarkLeasingPayment(context.Background(), "c1", "administracija", "user-1", "v1",
		dto.LeasingPaymentRequest{PeriodMonth: "2026-09-01"})
	if err != nil {
		t.Fatalf("unexpected error marking leasing payment: %v", err)
	}
	if payment == nil {
		t.Fatal("expected a payment response")
	}
	if payment.PeriodMonth != "2026-09-01" {
		t.Errorf("expected period_month=2026-09-01, got: %s", payment.PeriodMonth)
	}
}

func TestCompanyAssets_LeasingHistory_RemainsAfterLeasingDisabled(t *testing.T) {
	pastPayment := repositories.LeasingPaymentRow{
		ID: "p1", AssetID: "v1",
		PeriodMonth: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		CompletedAt: time.Now(), CompletedBy: "u1",
	}
	repo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _, id string) (*repositories.CompanyAssetRow, error) {
			return &repositories.CompanyAssetRow{ID: id, AssetType: "vozilo", IsLeasing: false}, nil
		},
		getLeasingPaymentsByAssetFn: func(_ context.Context, _, _ string) ([]repositories.LeasingPaymentRow, error) {
			return []repositories.LeasingPaymentRow{pastPayment}, nil
		},
	}
	svc := newAssetSvc(repo)
	history, err := svc.GetLeasingHistory(context.Background(), "c1", "administracija", "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 history record even after leasing disabled, got %d", len(history))
	}
	if history[0].PeriodMonth != "2026-08-01" {
		t.Errorf("expected period_month=2026-08-01, got: %s", history[0].PeriodMonth)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func filterLeasing(notes []dto.AssetNotification) []dto.AssetNotification {
	var out []dto.AssetNotification
	for _, n := range notes {
		if n.Kind == dto.NotificationKindLeasingWarning || n.Kind == dto.NotificationKindLeasingOverdue {
			out = append(out, n)
		}
	}
	return out
}
