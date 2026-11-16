package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Repo interface ────────────────────────────────────────────────────────────

type revenueRepoIface interface {
	CreateContract(ctx context.Context, p repositories.CreateContractParams) (*repositories.ProjectContractRow, error)
	GetContract(ctx context.Context, companyID, contractID string) (*repositories.ProjectContractRow, error)
	ListContractsForProject(ctx context.Context, companyID, projectID string) ([]repositories.ProjectContractRow, error)
	UpdateContract(ctx context.Context, p repositories.UpdateContractParams) (*repositories.ProjectContractRow, error)
	DeleteContract(ctx context.Context, companyID, contractID string) error

	CreateInvoice(ctx context.Context, p repositories.CreateProjectInvoiceParams) (*repositories.ProjectInvoiceRow, error)
	GetInvoice(ctx context.Context, companyID, invoiceID string) (*repositories.ProjectInvoiceRow, error)
	ListInvoices(ctx context.Context, companyID string, projectID, status *string, year, month int) ([]repositories.ProjectInvoiceRow, error)
	CancelInvoice(ctx context.Context, companyID, invoiceID string, notes *string) error

	AddPayment(ctx context.Context, p repositories.AddPaymentParams) (*repositories.InvoicePaymentRow, error)
	ListPayments(ctx context.Context, companyID, invoiceID string) ([]repositories.InvoicePaymentRow, error)

	GetProjectRevenueSummaries(ctx context.Context, companyID string, year, month int) ([]repositories.ProjectRevenueSummaryRow, error)
	GetCompanyMonthlyRevenueTrend(ctx context.Context, companyID string, year int) ([]repositories.MonthlyRevenueTrendRow, error)
	GetProjectMonthlyRevenueTrend(ctx context.Context, companyID, projectID string, year int) ([]repositories.MonthlyRevenueTrendRow, error)
	GetContractTotalForProject(ctx context.Context, companyID, projectID string) (float64, error)
}

// ── Service ───────────────────────────────────────────────────────────────────

type RevenueService struct {
	repo revenueRepoIface
}

func NewRevenueService(repo *repositories.RevenueRepository) *RevenueService {
	return &RevenueService{repo: repo}
}

// ── Access control ────────────────────────────────────────────────────────────

// requireRevenueWriteAccess restricts financial mutations to direktor only.
func requireRevenueWriteAccess(role string) error {
	if role != "direktor" {
		return ErrForbidden
	}
	return nil
}

// ── Contracts ─────────────────────────────────────────────────────────────────

func (s *RevenueService) CreateContract(ctx context.Context, companyID, callerRole, callerUserID string, req dto.CreateContractReq) (*dto.ProjectContract, error) {
	if err := requireRevenueWriteAccess(callerRole); err != nil {
		return nil, err
	}
	if req.ProjectID == "" {
		return nil, validationErr("ID projekta je obavezan")
	}
	if req.ContractType != "initial" && req.ContractType != "amendment" {
		return nil, validationErr("Vrsta ugovora mora biti 'initial' ili 'amendment'")
	}
	if req.NetValue == 0 && req.ContractType == "initial" {
		return nil, validationErr("Vrijednost ugovora mora biti različita od nule")
	}

	var contractDate *time.Time
	if req.ContractDate != nil && *req.ContractDate != "" {
		t, err := time.Parse("2006-01-02", *req.ContractDate)
		if err != nil {
			return nil, validationErr("Neispravan format datuma ugovora (YYYY-MM-DD)")
		}
		contractDate = &t
	}

	row, err := s.repo.CreateContract(ctx, repositories.CreateContractParams{
		CompanyID:    companyID,
		ProjectID:    req.ProjectID,
		ContractType: req.ContractType,
		Reference:    req.Reference,
		ContractDate: contractDate,
		NetValue:     req.NetValue,
		Notes:        req.Notes,
		CreatedBy:    callerUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("revenue.CreateContract: %w", err)
	}
	return contractRowToDTO(row), nil
}

func (s *RevenueService) GetContract(ctx context.Context, companyID, callerRole, contractID string) (*dto.ProjectContract, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	row, err := s.repo.GetContract(ctx, companyID, contractID)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("revenue.GetContract: %w", err)
	}
	return contractRowToDTO(row), nil
}

func (s *RevenueService) ListContractsForProject(ctx context.Context, companyID, callerRole, projectID string) ([]dto.ProjectContract, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListContractsForProject(ctx, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("revenue.ListContractsForProject: %w", err)
	}
	out := make([]dto.ProjectContract, 0, len(rows))
	for _, r := range rows {
		out = append(out, *contractRowToDTO(&r))
	}
	return out, nil
}

func (s *RevenueService) UpdateContract(ctx context.Context, companyID, callerRole, contractID string, req dto.UpdateContractReq) (*dto.ProjectContract, error) {
	if err := requireRevenueWriteAccess(callerRole); err != nil {
		return nil, err
	}
	if req.ContractType != "initial" && req.ContractType != "amendment" {
		return nil, validationErr("Vrsta ugovora mora biti 'initial' ili 'amendment'")
	}

	var contractDate *time.Time
	if req.ContractDate != nil && *req.ContractDate != "" {
		t, err := time.Parse("2006-01-02", *req.ContractDate)
		if err != nil {
			return nil, validationErr("Neispravan format datuma ugovora (YYYY-MM-DD)")
		}
		contractDate = &t
	}

	row, err := s.repo.UpdateContract(ctx, repositories.UpdateContractParams{
		CompanyID:    companyID,
		ContractID:   contractID,
		ContractType: req.ContractType,
		Reference:    req.Reference,
		ContractDate: contractDate,
		NetValue:     req.NetValue,
		Notes:        req.Notes,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("revenue.UpdateContract: %w", err)
	}
	return contractRowToDTO(row), nil
}

func (s *RevenueService) DeleteContract(ctx context.Context, companyID, callerRole, contractID string) error {
	if err := requireRevenueWriteAccess(callerRole); err != nil {
		return err
	}
	if err := s.repo.DeleteContract(ctx, companyID, contractID); err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("revenue.DeleteContract: %w", err)
	}
	return nil
}

// ── Invoices ──────────────────────────────────────────────────────────────────

func (s *RevenueService) CreateInvoice(ctx context.Context, companyID, callerRole, callerUserID string, req dto.CreateProjectInvoiceReq) (*dto.ProjectInvoice, error) {
	if err := requireRevenueWriteAccess(callerRole); err != nil {
		return nil, err
	}
	if req.ProjectID == "" {
		return nil, validationErr("ID projekta je obavezan")
	}
	if req.InvoiceNumber == "" {
		return nil, validationErr("Broj računa je obavezan")
	}
	if req.NetAmount <= 0 {
		return nil, validationErr("Neto iznos mora biti veći od nule")
	}
	if req.VatAmount < 0 {
		return nil, validationErr("PDV iznos ne može biti negativan")
	}

	invoiceDate, err := time.Parse("2006-01-02", req.InvoiceDate)
	if err != nil {
		return nil, validationErr("Neispravan format datuma računa (YYYY-MM-DD)")
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return nil, validationErr("Neispravan format roka plaćanja (YYYY-MM-DD)")
		}
		dueDate = &t
	}

	row, err := s.repo.CreateInvoice(ctx, repositories.CreateProjectInvoiceParams{
		CompanyID:     companyID,
		ProjectID:     req.ProjectID,
		InvoiceNumber: req.InvoiceNumber,
		InvoiceDate:   invoiceDate,
		DueDate:       dueDate,
		NetAmount:     req.NetAmount,
		VatAmount:     req.VatAmount,
		Notes:         req.Notes,
		CreatedBy:     callerUserID,
	})
	if err != nil {
		if isDuplicateError(err) {
			return nil, validationErr("Račun s tim brojem već postoji")
		}
		return nil, fmt.Errorf("revenue.CreateInvoice: %w", err)
	}
	return invoiceRowToDTO(row), nil
}

func (s *RevenueService) GetInvoice(ctx context.Context, companyID, callerRole, invoiceID string) (*dto.ProjectInvoice, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	row, err := s.repo.GetInvoice(ctx, companyID, invoiceID)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("revenue.GetInvoice: %w", err)
	}
	return invoiceRowToDTO(row), nil
}

func (s *RevenueService) ListInvoices(ctx context.Context, companyID, callerRole string, projectID, status *string, year, month int) ([]dto.ProjectInvoice, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListInvoices(ctx, companyID, projectID, status, year, month)
	if err != nil {
		return nil, fmt.Errorf("revenue.ListInvoices: %w", err)
	}
	out := make([]dto.ProjectInvoice, 0, len(rows))
	for _, r := range rows {
		out = append(out, *invoiceRowToDTO(&r))
	}
	return out, nil
}

func (s *RevenueService) CancelInvoice(ctx context.Context, companyID, callerRole, invoiceID string, notes *string) error {
	if err := requireRevenueWriteAccess(callerRole); err != nil {
		return err
	}
	if err := s.repo.CancelInvoice(ctx, companyID, invoiceID, notes); err != nil {
		if isNotFound(err) {
			return validationErr("Račun nije pronađen ili je već otkazan")
		}
		return fmt.Errorf("revenue.CancelInvoice: %w", err)
	}
	return nil
}

// ── Payments ──────────────────────────────────────────────────────────────────

func (s *RevenueService) AddPayment(ctx context.Context, companyID, callerRole, callerUserID, invoiceID string, req dto.AddPaymentReq) (*dto.InvoicePayment, error) {
	if err := requireRevenueWriteAccess(callerRole); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, validationErr("Iznos plaćanja mora biti veći od nule")
	}
	paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
	if err != nil {
		return nil, validationErr("Neispravan format datuma plaćanja (YYYY-MM-DD)")
	}

	row, err := s.repo.AddPayment(ctx, repositories.AddPaymentParams{
		CompanyID:   companyID,
		InvoiceID:   invoiceID,
		PaymentDate: paymentDate,
		Amount:      req.Amount,
		Notes:       req.Notes,
		CreatedBy:   callerUserID,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("revenue.AddPayment: %w", err)
	}
	return paymentRowToDTO(row), nil
}

func (s *RevenueService) ListPayments(ctx context.Context, companyID, callerRole, invoiceID string) ([]dto.InvoicePayment, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListPayments(ctx, companyID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("revenue.ListPayments: %w", err)
	}
	out := make([]dto.InvoicePayment, 0, len(rows))
	for _, r := range rows {
		out = append(out, *paymentRowToDTO(&r))
	}
	return out, nil
}

// ── Analytics ─────────────────────────────────────────────────────────────────

func (s *RevenueService) GetRevenueSummary(ctx context.Context, companyID, callerRole string, year, month int) (*dto.ProjectRevenueSummary, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if year <= 0 {
		year = time.Now().Year()
	}

	projects, err := s.repo.GetProjectRevenueSummaries(ctx, companyID, year, month)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetRevenueSummary projects: %w", err)
	}

	trend, err := s.repo.GetCompanyMonthlyRevenueTrend(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetRevenueSummary trend: %w", err)
	}

	summary := &dto.ProjectRevenueSummary{Year: year, Month: month}
	for _, p := range projects {
		summary.TotalContracted += p.ContractValue
		summary.TotalInvoicedNet += p.InvoicedNet
		summary.TotalInvoicedGross += p.InvoicedGross
		summary.TotalCollected += p.Collected
		summary.TotalOutstanding += p.Outstanding
		if p.HasOverdue {
			summary.OverdueCount++
			summary.TotalOverdue += p.Outstanding
		}
		summary.Projects = append(summary.Projects, dto.ProjectRevenueRow{
			ProjectID:     p.ProjectID,
			ProjectName:   p.ProjectName,
			ContractValue: p.ContractValue,
			InvoicedNet:   p.InvoicedNet,
			InvoicedGross: p.InvoicedGross,
			Collected:     p.Collected,
			Outstanding:   p.Outstanding,
			InvoiceCount:  p.InvoiceCount,
			HasOverdue:    p.HasOverdue,
		})
	}

	for _, t := range trend {
		summary.MonthlyTrend = append(summary.MonthlyTrend, dto.MonthlyRevenueTrend{
			Month:         t.Month,
			InvoicedNet:   t.InvoicedNet,
			InvoicedGross: t.InvoicedGross,
			Collected:     t.Collected,
			HasData:       t.InvoicedGross > 0 || t.Collected > 0,
		})
	}

	if summary.Projects == nil {
		summary.Projects = []dto.ProjectRevenueRow{}
	}
	if summary.MonthlyTrend == nil {
		summary.MonthlyTrend = []dto.MonthlyRevenueTrend{}
	}
	return summary, nil
}

func (s *RevenueService) GetProjectRevenueDetail(ctx context.Context, companyID, callerRole, projectID string, year int) (*dto.ProjectRevenueDetail, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if year <= 0 {
		year = time.Now().Year()
	}

	contracts, err := s.repo.ListContractsForProject(ctx, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetProjectRevenueDetail contracts: %w", err)
	}

	invoices, err := s.repo.ListInvoices(ctx, companyID, &projectID, nil, year, 0)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetProjectRevenueDetail invoices: %w", err)
	}

	trend, err := s.repo.GetProjectMonthlyRevenueTrend(ctx, companyID, projectID, year)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetProjectRevenueDetail trend: %w", err)
	}

	contractTotal, err := s.repo.GetContractTotalForProject(ctx, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("revenue.GetProjectRevenueDetail contractTotal: %w", err)
	}

	detail := &dto.ProjectRevenueDetail{
		ProjectID:     projectID,
		Year:          year,
		ContractValue: contractTotal,
	}

	if len(contracts) > 0 {
		detail.ProjectName = contracts[0].ProjectName
	}

	for _, c := range contracts {
		detail.Contracts = append(detail.Contracts, *contractRowToDTO(&c))
	}

	for _, inv := range invoices {
		d := invoiceRowToDTO(&inv)
		detail.Invoices = append(detail.Invoices, *d)
		if inv.Status != "cancelled" {
			detail.InvoicedNet += inv.NetAmount
			detail.InvoicedGross += inv.GrossAmount
			detail.Collected += inv.PaidAmount
		}
	}
	detail.Outstanding = detail.InvoicedGross - detail.Collected

	for _, t := range trend {
		detail.MonthlyTrend = append(detail.MonthlyTrend, dto.MonthlyRevenueTrend{
			Month:         t.Month,
			InvoicedNet:   t.InvoicedNet,
			InvoicedGross: t.InvoicedGross,
			Collected:     t.Collected,
			HasData:       t.InvoicedGross > 0 || t.Collected > 0,
		})
	}

	if detail.Contracts == nil {
		detail.Contracts = []dto.ProjectContract{}
	}
	if detail.Invoices == nil {
		detail.Invoices = []dto.ProjectInvoice{}
	}
	if detail.MonthlyTrend == nil {
		detail.MonthlyTrend = []dto.MonthlyRevenueTrend{}
	}
	return detail, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isNotFound(err error) bool {
	return err == pgx.ErrNoRows
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}

func contractRowToDTO(r *repositories.ProjectContractRow) *dto.ProjectContract {
	c := &dto.ProjectContract{
		ID:           r.ID,
		ProjectID:    r.ProjectID,
		ProjectName:  r.ProjectName,
		ContractType: r.ContractType,
		Reference:    r.Reference,
		NetValue:     r.NetValue,
		Notes:        r.Notes,
		CreatedAt:    r.CreatedAt,
	}
	if r.ContractDate != nil {
		s := r.ContractDate.Format("2006-01-02")
		c.ContractDate = &s
	}
	return c
}

func invoiceRowToDTO(r *repositories.ProjectInvoiceRow) *dto.ProjectInvoice {
	inv := &dto.ProjectInvoice{
		ID:                r.ID,
		ProjectID:         r.ProjectID,
		ProjectName:       r.ProjectName,
		InvoiceNumber:     r.InvoiceNumber,
		InvoiceDate:       r.InvoiceDate.Format("2006-01-02"),
		NetAmount:         r.NetAmount,
		VatAmount:         r.VatAmount,
		GrossAmount:       r.GrossAmount,
		PaidAmount:        r.PaidAmount,
		OutstandingAmount: r.GrossAmount - r.PaidAmount,
		Status:            r.Status,
		CancelsInvoiceID:  r.CancelsInvoiceID,
		Notes:             r.Notes,
		CreatedAt:         r.CreatedAt,
	}
	if r.DueDate != nil {
		s := r.DueDate.Format("2006-01-02")
		inv.DueDate = &s
		inv.IsOverdue = (r.Status == "issued" || r.Status == "partial") && r.DueDate.Before(time.Now().Truncate(24*time.Hour))
	}
	return inv
}

func paymentRowToDTO(r *repositories.InvoicePaymentRow) *dto.InvoicePayment {
	return &dto.InvoicePayment{
		ID:          r.ID,
		InvoiceID:   r.InvoiceID,
		PaymentDate: r.PaymentDate.Format("2006-01-02"),
		Amount:      r.Amount,
		Notes:       r.Notes,
		CreatedAt:   r.CreatedAt,
	}
}
