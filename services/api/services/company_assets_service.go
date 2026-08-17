package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Repository interface ──────────────────────────────────────────────────────

type companyAssetsRepoIface interface {
	List(ctx context.Context, companyID string, f repositories.CompanyAssetsFilter) ([]repositories.CompanyAssetRow, error)
	GetByID(ctx context.Context, companyID, id string) (*repositories.CompanyAssetRow, error)
	Create(ctx context.Context, companyID string, row repositories.CompanyAssetRow) (string, error)
	Update(ctx context.Context, companyID, id string, row repositories.CompanyAssetRow) error
	Deactivate(ctx context.Context, companyID, id string) error
	ListActiveEmployees(ctx context.Context, companyID string) ([]repositories.AssetEmployeeRow, error)
	EmployeeBelongsToCompany(ctx context.Context, companyID, employeeID string) (bool, error)
	// Leasing payments
	CreateLeasingPayment(ctx context.Context, row repositories.LeasingPaymentRow) (*repositories.LeasingPaymentRow, error)
	ListCompletedLeasingForPeriod(ctx context.Context, companyID string, periodMonth time.Time) (map[string]bool, error)
	GetLeasingPaymentsByAsset(ctx context.Context, companyID, assetID string) ([]repositories.LeasingPaymentRow, error)
}

// ── Service ───────────────────────────────────────────────────────────────────

type CompanyAssetsService struct {
	repo companyAssetsRepoIface
}

func NewCompanyAssetsService(repo companyAssetsRepoIface) *CompanyAssetsService {
	return &CompanyAssetsService{repo: repo}
}

func (s *CompanyAssetsService) requireAdmin(role string) error {
	if role != "administracija" {
		return ErrForbidden
	}
	return nil
}

// ── List ──────────────────────────────────────────────────────────────────────

func (s *CompanyAssetsService) List(ctx context.Context, companyID, callerRole string, f dto.CompanyAssetFilter) ([]dto.CompanyAsset, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, companyID, repositories.CompanyAssetsFilter{
		AssetType:  f.AssetType,
		EmployeeID: f.EmployeeID,
		Status:     f.Status,
		Search:     f.Search,
	})
	if err != nil {
		return nil, fmt.Errorf("list company assets: %w", err)
	}
	out := make([]dto.CompanyAsset, len(rows))
	for i, r := range rows {
		out[i] = toAssetDTO(r)
	}
	return out, nil
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (s *CompanyAssetsService) GetByID(ctx context.Context, companyID, callerRole, id string) (*dto.CompanyAsset, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	row, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	a := toAssetDTO(*row)
	return &a, nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func (s *CompanyAssetsService) Create(ctx context.Context, companyID, callerRole string, req dto.CreateCompanyAssetRequest) (*dto.CompanyAsset, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}

	if err := validateAssetType(req.AssetType); err != nil {
		return nil, err
	}

	// Default size to M for oprema.
	if req.AssetType == dto.AssetTypeOprema {
		if req.Size == nil || *req.Size == "" {
			m := "M"
			req.Size = &m
		} else if !dto.ValidAssetSizes[*req.Size] {
			return nil, validationErr("Neispravna veličina opreme")
		}
	}

	row := repositories.CompanyAssetRow{
		AssetType: req.AssetType,
		Name:      req.Name,
		Status:    "active",
		Notes:     req.Notes,
		Size:      req.Size,
	}

	if req.AssignedEmployeeID != nil && *req.AssignedEmployeeID != "" {
		ok, err := s.repo.EmployeeBelongsToCompany(ctx, companyID, *req.AssignedEmployeeID)
		if err != nil {
			return nil, fmt.Errorf("create company asset: %w", err)
		}
		if !ok {
			return nil, validationErr("Zaposlenik ne pripada ovoj tvrtki")
		}
		row.AssignedEmployeeID = req.AssignedEmployeeID
	}

	if err := parseDateField(req.PurchasedAt, &row.PurchasedAt); err != nil {
		return nil, validationErr("Neispravan format datuma kupnje")
	}
	if err := parseDateField(req.WarrantyExpiresAt, &row.WarrantyExpiresAt); err != nil {
		return nil, validationErr("Neispravan format datuma garancije")
	}

	if req.AssetType == dto.AssetTypeVozilo {
		row.RegistrationPlate = req.RegistrationPlate
		if err := parseDateField(req.RegistrationDate, &row.RegistrationDate); err != nil {
			return nil, validationErr("Neispravan format datuma registracije")
		}
		if err := parseDateField(req.RegistrationExpiresAt, &row.RegistrationExpiresAt); err != nil {
			return nil, validationErr("Neispravan format datuma isteka registracije")
		}
		if err := validateRegDates(row.RegistrationDate, row.RegistrationExpiresAt); err != nil {
			return nil, err
		}
		row.IsLeasing = req.IsLeasing
		row.LeasingCompany = req.LeasingCompany
		if err := parseDateField(req.LeasingEndDate, &row.LeasingEndDate); err != nil {
			return nil, validationErr("Neispravan format datuma završetka leasinga")
		}
	}

	id, err := s.repo.Create(ctx, companyID, row)
	if err != nil {
		return nil, fmt.Errorf("create company asset: %w", err)
	}

	created, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("create company asset: refetch: %w", err)
	}
	a := toAssetDTO(*created)
	return &a, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (s *CompanyAssetsService) Update(ctx context.Context, companyID, callerRole, id string, req dto.UpdateCompanyAssetRequest) (*dto.CompanyAsset, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}

	status := existing.Status
	if req.Status != "" {
		if req.Status != "active" && req.Status != "inactive" {
			return nil, validationErr("Neispravan status")
		}
		status = req.Status
	}

	if existing.AssetType == dto.AssetTypeOprema {
		if req.Size == nil || *req.Size == "" {
			m := "M"
			req.Size = &m
		} else if !dto.ValidAssetSizes[*req.Size] {
			return nil, validationErr("Neispravna veličina opreme")
		}
	}

	row := repositories.CompanyAssetRow{
		Name:   req.Name,
		Status: status,
		Notes:  req.Notes,
		Size:   req.Size,
	}

	if req.AssignedEmployeeID != nil && *req.AssignedEmployeeID != "" {
		ok, err := s.repo.EmployeeBelongsToCompany(ctx, companyID, *req.AssignedEmployeeID)
		if err != nil {
			return nil, fmt.Errorf("update company asset: %w", err)
		}
		if !ok {
			return nil, validationErr("Zaposlenik ne pripada ovoj tvrtki")
		}
		row.AssignedEmployeeID = req.AssignedEmployeeID
	}

	if err := parseDateField(req.PurchasedAt, &row.PurchasedAt); err != nil {
		return nil, validationErr("Neispravan format datuma kupnje")
	}
	if err := parseDateField(req.WarrantyExpiresAt, &row.WarrantyExpiresAt); err != nil {
		return nil, validationErr("Neispravan format datuma garancije")
	}

	if existing.AssetType == dto.AssetTypeVozilo {
		row.RegistrationPlate = req.RegistrationPlate
		if err := parseDateField(req.RegistrationDate, &row.RegistrationDate); err != nil {
			return nil, validationErr("Neispravan format datuma registracije")
		}
		if err := parseDateField(req.RegistrationExpiresAt, &row.RegistrationExpiresAt); err != nil {
			return nil, validationErr("Neispravan format datuma isteka registracije")
		}
		if err := validateRegDates(row.RegistrationDate, row.RegistrationExpiresAt); err != nil {
			return nil, err
		}
		row.IsLeasing = req.IsLeasing
		row.LeasingCompany = req.LeasingCompany
		if err := parseDateField(req.LeasingEndDate, &row.LeasingEndDate); err != nil {
			return nil, validationErr("Neispravan format datuma završetka leasinga")
		}
	}

	if err := s.repo.Update(ctx, companyID, id, row); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("update company asset: refetch: %w", err)
	}
	a := toAssetDTO(*updated)
	return &a, nil
}

// ── Deactivate ────────────────────────────────────────────────────────────────

func (s *CompanyAssetsService) Deactivate(ctx context.Context, companyID, callerRole, id string) error {
	if err := s.requireAdmin(callerRole); err != nil {
		return err
	}
	return s.repo.Deactivate(ctx, companyID, id)
}

// ── FormData (employee picker) ────────────────────────────────────────────────

func (s *CompanyAssetsService) ListEmployees(ctx context.Context, companyID, callerRole string) ([]dto.AssetEmployee, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListActiveEmployees(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("list employees for assets: %w", err)
	}
	out := make([]dto.AssetEmployee, len(rows))
	for i, r := range rows {
		out[i] = dto.AssetEmployee{
			ID:        r.ID,
			FirstName: r.FirstName,
			LastName:  r.LastName,
			Role:      r.Role,
		}
	}
	return out, nil
}

// ── Notifications ─────────────────────────────────────────────────────────────

func (s *CompanyAssetsService) GetNotifications(ctx context.Context, companyID, callerRole string, today time.Time) ([]dto.AssetNotification, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, companyID, repositories.CompanyAssetsFilter{Status: "active"})
	if err != nil {
		return nil, fmt.Errorf("get notifications: %w", err)
	}

	// Fetch completed leasing months for the current period.
	periodMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	completedLeasing, err := s.repo.ListCompletedLeasingForPeriod(ctx, companyID, periodMonth)
	if err != nil {
		return nil, fmt.Errorf("get notifications: leasing: %w", err)
	}

	return ComputeNotifications(rows, today, completedLeasing), nil
}

// ComputeNotifications is exported for unit testing.
// completedLeasing is a set of asset IDs with a completed leasing payment for the current period month.
func ComputeNotifications(rows []repositories.CompanyAssetRow, today time.Time, completedLeasing map[string]bool) []dto.AssetNotification {
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	var out []dto.AssetNotification

	for _, r := range rows {
		var emp *dto.AssetEmployee
		if r.AssignedEmployeeID != nil && r.AssignedFirstName != nil {
			emp = &dto.AssetEmployee{
				ID:        *r.AssignedEmployeeID,
				FirstName: *r.AssignedFirstName,
				LastName:  ptr(r.AssignedLastName),
				Role:      ptr(r.AssignedRole),
			}
		}

		switch r.AssetType {
		case dto.AssetTypeAlat, dto.AssetTypeOprema:
			if r.WarrantyExpiresAt != nil {
				n := buildNotification(r.ID, r.Name, r.AssetType, dto.NotificationKindWarranty, *r.WarrantyExpiresAt, todayDate, emp)
				if n != nil {
					out = append(out, *n)
				}
			}
		case dto.AssetTypeVozilo:
			if r.RegistrationExpiresAt != nil {
				n := buildNotification(r.ID, r.Name, r.AssetType, dto.NotificationKindRegistration, *r.RegistrationExpiresAt, todayDate, emp)
				if n != nil {
					out = append(out, *n)
				}
			}
			// Leasing reminder: independent of registration.
			if r.IsLeasing {
				if n := buildLeasingNotification(r, todayDate, completedLeasing, emp); n != nil {
					out = append(out, *n)
				}
			}
		}
	}
	return out
}

// ── Leasing payments ──────────────────────────────────────────────────────────

func (s *CompanyAssetsService) MarkLeasingPayment(ctx context.Context, companyID, callerRole, callerUserID, assetID string, req dto.LeasingPaymentRequest) (*dto.LeasingPayment, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}

	// GetByID validates company ownership.
	asset, err := s.repo.GetByID(ctx, companyID, assetID)
	if err != nil {
		return nil, err
	}
	if asset.AssetType != dto.AssetTypeVozilo {
		return nil, validationErr("Leasing plaćanja su dostupna samo za vozila")
	}
	if !asset.IsLeasing {
		return nil, validationErr("Ovo vozilo nije na leasingu")
	}

	// Parse and normalise to first of month.
	periodMonth, err := time.Parse("2006-01-02", req.PeriodMonth)
	if err != nil {
		return nil, validationErr("Neispravan format datuma perioda (očekivano: YYYY-MM-01)")
	}
	normalized := time.Date(periodMonth.Year(), periodMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	if !normalized.Equal(periodMonth) {
		return nil, validationErr("Datum perioda mora biti prvi dan u mjesecu (npr. 2026-09-01)")
	}

	// Do not allow completing a month after the leasing contract ended.
	if asset.LeasingEndDate != nil {
		endDate := time.Date(asset.LeasingEndDate.Year(), asset.LeasingEndDate.Month(), asset.LeasingEndDate.Day(), 0, 0, 0, 0, time.UTC)
		if normalized.After(endDate) {
			return nil, validationErr("Leasing ugovor je završio prije ovog perioda")
		}
	}

	payment, err := s.repo.CreateLeasingPayment(ctx, repositories.LeasingPaymentRow{
		CompanyID:   companyID,
		AssetID:     assetID,
		PeriodMonth: normalized,
		CompletedBy: callerUserID,
	})
	if errors.Is(err, repositories.ErrDuplicateLeasingPayment) {
		return nil, validationErr("Plaćanje za ovaj mjesec već je zabilježeno")
	}
	if err != nil {
		return nil, fmt.Errorf("mark leasing payment: %w", err)
	}

	return &dto.LeasingPayment{
		ID:          payment.ID,
		AssetID:     payment.AssetID,
		PeriodMonth: payment.PeriodMonth.Format("2006-01-02"),
		CompletedAt: payment.CompletedAt,
		CompletedBy: payment.CompletedBy,
		CreatedAt:   payment.CreatedAt,
	}, nil
}

func (s *CompanyAssetsService) GetLeasingHistory(ctx context.Context, companyID, callerRole, assetID string) ([]dto.LeasingPayment, error) {
	if err := s.requireAdmin(callerRole); err != nil {
		return nil, err
	}
	// GetByID validates company ownership.
	if _, err := s.repo.GetByID(ctx, companyID, assetID); err != nil {
		return nil, err
	}
	rows, err := s.repo.GetLeasingPaymentsByAsset(ctx, companyID, assetID)
	if err != nil {
		return nil, fmt.Errorf("get leasing history: %w", err)
	}
	out := make([]dto.LeasingPayment, len(rows))
	for i, r := range rows {
		out[i] = dto.LeasingPayment{
			ID:          r.ID,
			AssetID:     r.AssetID,
			PeriodMonth: r.PeriodMonth.Format("2006-01-02"),
			CompletedAt: r.CompletedAt,
			CompletedBy: r.CompletedBy,
			CreatedAt:   r.CreatedAt,
		}
	}
	return out, nil
}

// ── Notification helpers ──────────────────────────────────────────────────────

func buildNotification(assetID, name, assetType, kind string, expiresAt, today time.Time, emp *dto.AssetEmployee) *dto.AssetNotification {
	expiryDate := time.Date(expiresAt.Year(), expiresAt.Month(), expiresAt.Day(), 0, 0, 0, 0, time.UTC)
	days := int(expiryDate.Sub(today).Hours() / 24)

	if days > 10 {
		return nil
	}

	var state string
	switch {
	case days < 0:
		state = dto.NotificationStateExpired
	case days == 0:
		state = dto.NotificationStateUrgent
	default:
		state = dto.NotificationStateWarning
	}

	expiresStr := expiresAt.Format("2006-01-02")
	msg := buildMessage(kind, days)

	return &dto.AssetNotification{
		AssetID:          assetID,
		AssetName:        name,
		AssetType:        assetType,
		Kind:             kind,
		State:            state,
		DaysRemaining:    days,
		ExpiresAt:        expiresStr,
		AssignedEmployee: emp,
		Message:          msg,
	}
}

// Croatian month names (nominative).
var hrMonths = [13]string{
	"", "siječanj", "veljača", "ožujak", "travanj", "svibanj", "lipanj",
	"srpanj", "kolovoz", "rujan", "listopad", "studeni", "prosinac",
}

func buildLeasingNotification(r repositories.CompanyAssetRow, today time.Time, completedLeasing map[string]bool, emp *dto.AssetEmployee) *dto.AssetNotification {
	periodMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Skip if leasing contract has already ended before this period month.
	if r.LeasingEndDate != nil {
		endDate := time.Date(r.LeasingEndDate.Year(), r.LeasingEndDate.Month(), r.LeasingEndDate.Day(), 0, 0, 0, 0, time.UTC)
		if periodMonth.After(endDate) {
			return nil
		}
	}

	// Skip if already completed for the current period.
	if completedLeasing != nil && completedLeasing[r.ID] {
		return nil
	}

	dueDate := time.Date(today.Year(), today.Month(), 15, 0, 0, 0, 0, time.UTC)
	daysUntilDue := int(dueDate.Sub(today).Hours() / 24)

	var kind, state string
	if today.Day() <= 15 {
		kind = dto.NotificationKindLeasingWarning
		state = dto.NotificationStateWarning
	} else {
		kind = dto.NotificationKindLeasingOverdue
		state = dto.NotificationStateUrgent
	}

	return &dto.AssetNotification{
		AssetID:           r.ID,
		AssetName:         r.Name,
		AssetType:         r.AssetType,
		Kind:              kind,
		State:             state,
		DaysRemaining:     daysUntilDue,
		ExpiresAt:         dueDate.Format("2006-01-02"),
		AssignedEmployee:  emp,
		Message:           buildLeasingMessage(periodMonth, today.Day()),
		RegistrationPlate: r.RegistrationPlate,
		PeriodMonth:       periodMonth.Format("2006-01-02"),
	}
}

func buildLeasingMessage(periodMonth time.Time, dayOfMonth int) string {
	month := hrMonths[periodMonth.Month()]
	year := periodMonth.Year()
	dueDate := time.Date(year, periodMonth.Month(), 15, 0, 0, 0, 0, time.UTC)
	dueStr := dueDate.Format("02.01.2006.")

	if dayOfMonth <= 15 {
		return fmt.Sprintf("Leasing za %s %d. potrebno je riješiti do %s", month, year, dueStr)
	}
	return fmt.Sprintf("Leasing za %s %d. nije riješen. Rok je bio %s", month, year, dueStr)
}

func buildMessage(kind string, days int) string {
	var subject string
	if kind == dto.NotificationKindWarranty {
		subject = "Garancija"
	} else {
		subject = "Registracija"
	}

	switch {
	case days < 0:
		n := -days
		if n == 1 {
			return fmt.Sprintf("%s je istekla prije 1 dan", subject)
		}
		return fmt.Sprintf("%s je istekla prije %d dana", subject, n)
	case days == 0:
		return fmt.Sprintf("%s istječe danas", subject)
	case days == 1:
		return fmt.Sprintf("%s istječe za 1 dan", subject)
	default:
		return fmt.Sprintf("%s istječe za %d dana", subject, days)
	}
}

// ── DTO conversion ────────────────────────────────────────────────────────────

func toAssetDTO(r repositories.CompanyAssetRow) dto.CompanyAsset {
	a := dto.CompanyAsset{
		ID:                r.ID,
		CompanyID:         r.CompanyID,
		AssetType:         r.AssetType,
		Name:              r.Name,
		Status:            r.Status,
		Notes:             r.Notes,
		Size:              r.Size,
		RegistrationPlate: r.RegistrationPlate,
		IsLeasing:         r.IsLeasing,
		LeasingCompany:    r.LeasingCompany,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
	if r.AssignedEmployeeID != nil && r.AssignedFirstName != nil {
		a.AssignedEmployee = &dto.AssetEmployee{
			ID:        *r.AssignedEmployeeID,
			FirstName: *r.AssignedFirstName,
			LastName:  ptr(r.AssignedLastName),
			Role:      ptr(r.AssignedRole),
		}
	}
	if r.PurchasedAt != nil {
		s := r.PurchasedAt.Format("2006-01-02")
		a.PurchasedAt = &s
	}
	if r.WarrantyExpiresAt != nil {
		s := r.WarrantyExpiresAt.Format("2006-01-02")
		a.WarrantyExpiresAt = &s
	}
	if r.RegistrationDate != nil {
		s := r.RegistrationDate.Format("2006-01-02")
		a.RegistrationDate = &s
	}
	if r.RegistrationExpiresAt != nil {
		s := r.RegistrationExpiresAt.Format("2006-01-02")
		a.RegistrationExpiresAt = &s
	}
	if r.LeasingEndDate != nil {
		s := r.LeasingEndDate.Format("2006-01-02")
		a.LeasingEndDate = &s
	}
	return a
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func validateAssetType(t string) error {
	if !dto.ValidAssetTypes[t] {
		return validationErr("Neispravna vrsta imovine (alat, oprema, vozilo)")
	}
	return nil
}

func parseDateField(s *string, dest **time.Time) error {
	if s == nil || *s == "" {
		*dest = nil
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return err
	}
	*dest = &t
	return nil
}

func validateRegDates(regDate, expiresAt *time.Time) error {
	if regDate == nil || expiresAt == nil {
		return nil
	}
	if expiresAt.Before(*regDate) {
		return validationErr("Datum isteka registracije ne smije biti prije datuma registracije")
	}
	return nil
}

func ptr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
