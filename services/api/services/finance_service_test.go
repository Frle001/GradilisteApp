package services

import (
	"context"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

var errNotFoundPgx = pgx.ErrNoRows

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockFinanceRepo struct {
	invoices     []repositories.CompanyInvoiceRow
	r1s          []repositories.R1ReceiptRow
	createInvErr error
	createR1Err  error
}

func (m *mockFinanceRepo) CreateInvoice(_ context.Context, p repositories.CreateInvoiceParams) (*repositories.CompanyInvoiceRow, error) {
	if m.createInvErr != nil {
		return nil, m.createInvErr
	}
	row := repositories.CompanyInvoiceRow{
		ID:               "inv-1",
		InvoiceType:      p.InvoiceType,
		Supplier:         p.Supplier,
		LeasingCompany:   p.LeasingCompany,
		StorageKey:       p.StorageKey,
		OriginalFilename: p.OriginalFilename,
		MimeType:         p.MimeType,
		FileSize:         p.FileSize,
		CreatedBy:        p.CreatedBy,
		CreatedByName:    "Test User",
	}
	return &row, nil
}

func (m *mockFinanceRepo) ListInvoices(_ context.Context, _ string) ([]repositories.CompanyInvoiceRow, error) {
	return m.invoices, nil
}

func (m *mockFinanceRepo) GetInvoice(_ context.Context, companyID, invoiceID string) (*repositories.CompanyInvoiceRow, error) {
	for _, r := range m.invoices {
		if r.ID == invoiceID {
			return &r, nil
		}
	}
	return nil, errNotFoundPgx
}

func (m *mockFinanceRepo) CreateR1Receipt(_ context.Context, p repositories.CreateR1Params) (*repositories.R1ReceiptRow, error) {
	if m.createR1Err != nil {
		return nil, m.createR1Err
	}
	row := repositories.R1ReceiptRow{
		ID:               "r1-1",
		SubmittedBy:      p.SubmittedBy,
		SubmitterName:    "Test User",
		Price:            p.Price,
		StorageKey:       p.StorageKey,
		OriginalFilename: p.OriginalFilename,
		MimeType:         p.MimeType,
		FileSize:         p.FileSize,
	}
	return &row, nil
}

func (m *mockFinanceRepo) ListR1Receipts(_ context.Context, _ string, submittedBy *string) ([]repositories.R1ReceiptRow, error) {
	if submittedBy == nil {
		return m.r1s, nil
	}
	var out []repositories.R1ReceiptRow
	for _, r := range m.r1s {
		if r.SubmittedBy == *submittedBy {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockFinanceRepo) GetR1Receipt(_ context.Context, _, receiptID string) (*repositories.R1ReceiptRow, error) {
	for _, r := range m.r1s {
		if r.ID == receiptID {
			return &r, nil
		}
	}
	return nil, errNotFoundPgx
}

type mockFinanceStorage struct {
	savedSubdirs []string
	saveErr      error
}

func (m *mockFinanceStorage) SaveFinanceDocFile(_ context.Context, _ multipart.File, h *multipart.FileHeader, _, subdir string) (string, string, int64, error) {
	if m.saveErr != nil {
		return "", "", 0, m.saveErr
	}
	m.savedSubdirs = append(m.savedSubdirs, subdir)
	if h == nil {
		return "finance-docs/co/subdir/id.pdf", "application/pdf", 100, nil
	}
	return "finance-docs/co/" + subdir + "/id.pdf", "application/pdf", h.Size, nil
}

func (m *mockFinanceStorage) ReadFinanceDocFile(_ context.Context, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader("file content")), "application/pdf", nil
}

func newFinanceSvc(repo financeRepo, storage financeStorage) *FinanceService {
	return &FinanceService{repo: repo, storage: storage}
}

func invoiceSvc() *FinanceService {
	return newFinanceSvc(&mockFinanceRepo{}, &mockFinanceStorage{})
}

func dummyFileHeader() *multipart.FileHeader {
	return makeFileHeaderWithCT("invoice.pdf", "application/pdf", 1024)
}

type nopReadCloser struct{ *strings.Reader }

func (nopReadCloser) Close() error { return nil }

func dummyFile() multipart.File {
	return nopReadCloser{strings.NewReader("pdf content")}
}

// ── validateInvoiceFields ─────────────────────────────────────────────────────

func TestInvoice_InvalidType_Rejected(t *testing.T) {
	err := validateInvoiceFields("unknown", nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestInvoice_Materijal_ValidSupplier(t *testing.T) {
	sup := dto.SupplierEnergyCentar
	if err := validateInvoiceFields(dto.InvoiceTypeMaterijal, &sup, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestInvoice_Materijal_NoSupplier_Rejected(t *testing.T) {
	err := validateInvoiceFields(dto.InvoiceTypeMaterijal, nil, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for missing supplier, got %v", err)
	}
}

func TestInvoice_Materijal_InvalidSupplier_Rejected(t *testing.T) {
	sup := "Unknown Supplier"
	err := validateInvoiceFields(dto.InvoiceTypeMaterijal, &sup, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for invalid supplier, got %v", err)
	}
}

func TestInvoice_Materijal_WithLeasingCompany_Rejected(t *testing.T) {
	sup := dto.SupplierPondeljak
	lc := dto.LeasingImpuls
	err := validateInvoiceFields(dto.InvoiceTypeMaterijal, &sup, &lc)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for leasing_company on materijal, got %v", err)
	}
}

func TestInvoice_Leasing_ValidCompany(t *testing.T) {
	lc := dto.LeasingUnicreditLeasing
	if err := validateInvoiceFields(dto.InvoiceTypeLeasing, nil, &lc); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestInvoice_Leasing_NoCompany_Rejected(t *testing.T) {
	err := validateInvoiceFields(dto.InvoiceTypeLeasing, nil, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for missing leasing company, got %v", err)
	}
}

func TestInvoice_Leasing_InvalidCompany_Rejected(t *testing.T) {
	lc := "Unknown Leasing"
	err := validateInvoiceFields(dto.InvoiceTypeLeasing, nil, &lc)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for invalid leasing company, got %v", err)
	}
}

func TestInvoice_Leasing_WithSupplier_Rejected(t *testing.T) {
	sup := dto.SupplierLipaPromet
	lc := dto.LeasingImpuls
	err := validateInvoiceFields(dto.InvoiceTypeLeasing, &sup, &lc)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for supplier on leasing, got %v", err)
	}
}

func TestInvoice_Gorivo_NoExtraFields(t *testing.T) {
	if err := validateInvoiceFields(dto.InvoiceTypeGorivo, nil, nil); err != nil {
		t.Fatalf("expected no error for gorivo without extra fields, got %v", err)
	}
}

func TestInvoice_Gorivo_WithSupplier_Rejected(t *testing.T) {
	sup := dto.SupplierEnergyCentar
	err := validateInvoiceFields(dto.InvoiceTypeGorivo, &sup, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for supplier on gorivo, got %v", err)
	}
}

func TestInvoice_Gorivo_WithLeasingCompany_Rejected(t *testing.T) {
	lc := dto.LeasingImpuls
	err := validateInvoiceFields(dto.InvoiceTypeGorivo, nil, &lc)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for leasing_company on gorivo, got %v", err)
	}
}

func TestInvoice_Alati_Valid(t *testing.T) {
	if err := validateInvoiceFields(dto.InvoiceTypeAlati, nil, nil); err != nil {
		t.Fatalf("expected no error for alati, got %v", err)
	}
}

func TestInvoice_Oprema_Valid(t *testing.T) {
	if err := validateInvoiceFields(dto.InvoiceTypeOprema, nil, nil); err != nil {
		t.Fatalf("expected no error for oprema, got %v", err)
	}
}

// ── Role access for Računi ────────────────────────────────────────────────────

func TestRacuni_DirektorAllowed(t *testing.T) {
	if err := requireRacuniAccess("direktor"); err != nil {
		t.Fatalf("direktor should be allowed: %v", err)
	}
}

func TestRacuni_AdministracijaAllowed(t *testing.T) {
	if err := requireRacuniAccess("administracija"); err != nil {
		t.Fatalf("administracija should be allowed: %v", err)
	}
}

func TestRacuni_InzenjerForbidden(t *testing.T) {
	if err := requireRacuniAccess("inzenjer"); err != ErrForbidden {
		t.Fatalf("inzenjer should be forbidden, got %v", err)
	}
}

func TestRacuni_PoslovodaForbidden(t *testing.T) {
	if err := requireRacuniAccess("poslovoda"); err != ErrForbidden {
		t.Fatalf("poslovoda should be forbidden, got %v", err)
	}
}

func TestRacuni_RadnikForbidden(t *testing.T) {
	if err := requireRacuniAccess("radnik"); err != ErrForbidden {
		t.Fatalf("radnik should be forbidden, got %v", err)
	}
}

// ── Role access for R1 (create) ───────────────────────────────────────────────

func TestR1Create_DirektorAllowed(t *testing.T) {
	if err := requireR1CreateAccess("direktor"); err != nil {
		t.Fatalf("direktor should be allowed to create R1: %v", err)
	}
}

func TestR1Create_InzenjerAllowed(t *testing.T) {
	if err := requireR1CreateAccess("inzenjer"); err != nil {
		t.Fatalf("inzenjer should be allowed to create R1: %v", err)
	}
}

func TestR1Create_PoslovodaAllowed(t *testing.T) {
	if err := requireR1CreateAccess("poslovoda"); err != nil {
		t.Fatalf("poslovoda should be allowed to create R1: %v", err)
	}
}

func TestR1Create_RadnikAllowed(t *testing.T) {
	if err := requireR1CreateAccess("radnik"); err != nil {
		t.Fatalf("radnik should be allowed to create R1: %v", err)
	}
}

func TestR1Create_AdministracijaForbidden(t *testing.T) {
	if err := requireR1CreateAccess("administracija"); err != ErrForbidden {
		t.Fatalf("administracija should be forbidden from creating R1, got %v", err)
	}
}

// ── Role access for R1 (read) ─────────────────────────────────────────────────

func TestR1Read_AdministracijaAllowed(t *testing.T) {
	if err := requireR1ReadAccess("administracija"); err != nil {
		t.Fatalf("administracija should be allowed to read R1: %v", err)
	}
}

// ── CreateInvoice via service ─────────────────────────────────────────────────

func TestCreateInvoice_RoleCheck_ForbidInzenjer(t *testing.T) {
	svc := invoiceSvc()
	sup := dto.SupplierEnergyCentar
	_, err := svc.CreateInvoice(context.Background(), "co-1", "inzenjer", "u-1", dto.InvoiceTypeMaterijal, &sup, nil, nil, nil)
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateInvoice_MaterialWithoutSupplier_Rejected(t *testing.T) {
	svc := invoiceSvc()
	_, err := svc.CreateInvoice(context.Background(), "co-1", "direktor", "u-1", dto.InvoiceTypeMaterijal, nil, nil, nil, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestCreateInvoice_LeasingWithoutCompany_Rejected(t *testing.T) {
	svc := invoiceSvc()
	_, err := svc.CreateInvoice(context.Background(), "co-1", "administracija", "u-1", dto.InvoiceTypeLeasing, nil, nil, nil, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestCreateInvoice_MaterialValid_Succeeds(t *testing.T) {
	svc := invoiceSvc()
	sup := dto.SupplierPondeljak
	inv, err := svc.CreateInvoice(context.Background(), "co-1", "direktor", "u-1", dto.InvoiceTypeMaterijal, &sup, nil, dummyFile(), dummyFileHeader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.InvoiceType != dto.InvoiceTypeMaterijal {
		t.Errorf("expected type materijal, got %s", inv.InvoiceType)
	}
	if inv.Supplier == nil || *inv.Supplier != sup {
		t.Errorf("expected supplier %q, got %v", sup, inv.Supplier)
	}
}

func TestCreateInvoice_SubmittedByDerivedFromContext(t *testing.T) {
	repo := &mockFinanceRepo{}
	storage := &mockFinanceStorage{}
	svc := newFinanceSvc(repo, storage)
	sup := dto.SupplierEnergyCentar
	inv, err := svc.CreateInvoice(context.Background(), "co-1", "direktor", "user-from-jwt", dto.InvoiceTypeMaterijal, &sup, nil, dummyFile(), dummyFileHeader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.CreatedBy != "user-from-jwt" {
		t.Errorf("expected created_by from JWT, got %q", inv.CreatedBy)
	}
}

func TestCreateInvoice_StorageSubdirIsRacuni(t *testing.T) {
	storage := &mockFinanceStorage{}
	svc := newFinanceSvc(&mockFinanceRepo{}, storage)
	sup := dto.SupplierEnergyCentar
	_, err := svc.CreateInvoice(context.Background(), "co-1", "direktor", "u-1", dto.InvoiceTypeMaterijal, &sup, nil, dummyFile(), dummyFileHeader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(storage.savedSubdirs) == 0 || storage.savedSubdirs[0] != "racuni" {
		t.Errorf("expected subdir 'racuni', got %v", storage.savedSubdirs)
	}
}

// ── R1 price validation ───────────────────────────────────────────────────────

func TestCreateR1_ZeroPrice_Rejected(t *testing.T) {
	svc := newFinanceSvc(&mockFinanceRepo{}, &mockFinanceStorage{})
	_, err := svc.CreateR1Receipt(context.Background(), "co-1", "radnik", "u-1", 0, nil, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for zero price, got %v", err)
	}
}

func TestCreateR1_NegativePrice_Rejected(t *testing.T) {
	svc := newFinanceSvc(&mockFinanceRepo{}, &mockFinanceStorage{})
	_, err := svc.CreateR1Receipt(context.Background(), "co-1", "inzenjer", "u-1", -100, nil, nil)
	if AsValidationError(err) == nil {
		t.Fatalf("expected ValidationError for negative price, got %v", err)
	}
}

func TestCreateR1_AdministracijaForbidden(t *testing.T) {
	svc := newFinanceSvc(&mockFinanceRepo{}, &mockFinanceStorage{})
	_, err := svc.CreateR1Receipt(context.Background(), "co-1", "administracija", "u-1", 100, nil, nil)
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateR1_ValidPrice_Succeeds(t *testing.T) {
	svc := newFinanceSvc(&mockFinanceRepo{}, &mockFinanceStorage{})
	rec, err := svc.CreateR1Receipt(context.Background(), "co-1", "radnik", "u-1", 99.99, dummyFile(), dummyFileHeader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Price != 99.99 {
		t.Errorf("expected price 99.99, got %v", rec.Price)
	}
}

func TestCreateR1_SubmittedByFromContext(t *testing.T) {
	svc := newFinanceSvc(&mockFinanceRepo{}, &mockFinanceStorage{})
	rec, err := svc.CreateR1Receipt(context.Background(), "co-1", "inzenjer", "jwt-user-id", 50, dummyFile(), dummyFileHeader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.SubmittedBy != "jwt-user-id" {
		t.Errorf("expected submitted_by from JWT, got %q", rec.SubmittedBy)
	}
}

func TestCreateR1_StorageSubdirIsR1(t *testing.T) {
	storage := &mockFinanceStorage{}
	svc := newFinanceSvc(&mockFinanceRepo{}, storage)
	_, err := svc.CreateR1Receipt(context.Background(), "co-1", "poslovoda", "u-1", 25.50, dummyFile(), dummyFileHeader())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(storage.savedSubdirs) == 0 || storage.savedSubdirs[0] != "r1" {
		t.Errorf("expected subdir 'r1', got %v", storage.savedSubdirs)
	}
}

// ── R1 visibility ─────────────────────────────────────────────────────────────

func TestListR1_DirektorSeesAll(t *testing.T) {
	repo := &mockFinanceRepo{
		r1s: []repositories.R1ReceiptRow{
			{ID: "r1-1", SubmittedBy: "user-a", Price: 100},
			{ID: "r1-2", SubmittedBy: "user-b", Price: 200},
		},
	}
	svc := newFinanceSvc(repo, &mockFinanceStorage{})
	list, err := svc.ListR1Receipts(context.Background(), "co-1", "direktor", "user-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("direktor should see 2 receipts, got %d", len(list))
	}
}

func TestListR1_NonDirektorSeesOwnOnly(t *testing.T) {
	repo := &mockFinanceRepo{
		r1s: []repositories.R1ReceiptRow{
			{ID: "r1-1", SubmittedBy: "user-a", Price: 100},
			{ID: "r1-2", SubmittedBy: "user-b", Price: 200},
		},
	}
	svc := newFinanceSvc(repo, &mockFinanceStorage{})
	list, err := svc.ListR1Receipts(context.Background(), "co-1", "radnik", "user-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("radnik should see 1 receipt (own), got %d", len(list))
	}
	if list[0].SubmittedBy != "user-a" {
		t.Errorf("radnik sees wrong receipt: %v", list[0].SubmittedBy)
	}
}

func TestListR1_AdministracijaSeesAll(t *testing.T) {
	repo := &mockFinanceRepo{
		r1s: []repositories.R1ReceiptRow{
			{ID: "r1-1", SubmittedBy: "user-a", Price: 100},
			{ID: "r1-2", SubmittedBy: "user-b", Price: 200},
		},
	}
	svc := newFinanceSvc(repo, &mockFinanceStorage{})
	list, err := svc.ListR1Receipts(context.Background(), "co-1", "administracija", "admin-user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("administracija should see all 2 receipts, got %d", len(list))
	}
}

func TestDownloadR1_AdministracijaCanSeeOthers(t *testing.T) {
	repo := &mockFinanceRepo{
		r1s: []repositories.R1ReceiptRow{
			{ID: "r1-x", SubmittedBy: "user-b", StorageKey: "some/key"},
		},
	}
	svc := newFinanceSvc(repo, &mockFinanceStorage{})
	reader, _, _, err := svc.DownloadR1Receipt(context.Background(), "co-1", "administracija", "admin-user", "r1-x")
	if err != nil {
		t.Fatalf("administracija should access any same-company receipt, got %v", err)
	}
	if reader != nil {
		reader.Close()
	}
}

func TestDownloadR1_NonDirektor_OtherUserReceipt_Forbidden(t *testing.T) {
	repo := &mockFinanceRepo{
		r1s: []repositories.R1ReceiptRow{
			{ID: "r1-x", SubmittedBy: "user-b", StorageKey: "some/key"},
		},
	}
	svc := newFinanceSvc(repo, &mockFinanceStorage{})
	_, _, _, err := svc.DownloadR1Receipt(context.Background(), "co-1", "inzenjer", "user-a", "r1-x")
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden when non-director tries to download other's receipt, got %v", err)
	}
}

func TestDownloadR1_DirektorCanSeeOthers(t *testing.T) {
	repo := &mockFinanceRepo{
		r1s: []repositories.R1ReceiptRow{
			{ID: "r1-x", SubmittedBy: "user-b", StorageKey: "some/key"},
		},
	}
	svc := newFinanceSvc(repo, &mockFinanceStorage{})
	reader, _, _, err := svc.DownloadR1Receipt(context.Background(), "co-1", "direktor", "user-a", "r1-x")
	if err != nil {
		t.Fatalf("direktor should access others' receipts, got %v", err)
	}
	if reader != nil {
		reader.Close()
	}
}
