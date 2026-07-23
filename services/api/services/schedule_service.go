package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// scheduleRepoIface is the repository surface used by ScheduleService.
type scheduleRepoIface interface {
	ProjectBelongsToCompany(ctx context.Context, companyID, projectID string) (bool, error)
	CreateShift(ctx context.Context, tx pgx.Tx, companyID, projectID, shiftDate string, startTime, endTime *string, notes *string, createdBy string) (string, error)
	GetShift(ctx context.Context, companyID, shiftID string) (*repositories.ShiftRow, error)
	GetShiftForUpdate(ctx context.Context, tx pgx.Tx, companyID, shiftID string) (*repositories.ShiftRow, error)
	ListShifts(ctx context.Context, companyID, dateFrom, dateTo string, projectID *string) ([]repositories.ShiftRow, error)
	UpdateShift(ctx context.Context, tx pgx.Tx, shiftID, shiftDate string, startTime, endTime *string, notes *string) error
	CancelShift(ctx context.Context, tx pgx.Tx, companyID, shiftID, cancelledBy string) error
	ListAssignments(ctx context.Context, companyID, shiftID string) ([]repositories.ShiftAssignmentRow, error)
	DeleteAssignments(ctx context.Context, tx pgx.Tx, shiftID string) error
	CreateAssignment(ctx context.Context, tx pgx.Tx, companyID, shiftID, employeeID, assignedBy string, overlapOverridden bool, overriddenBy *string) error
	EmployeesForDate(ctx context.Context, companyID, shiftDate, excludeShiftID string) ([]repositories.EmployeeForDateRow, error)
}

// ScheduleService handles company shift scheduling.
type ScheduleService struct {
	db   txBeginner
	repo scheduleRepoIface
}

func NewScheduleService(db *pgxpool.Pool, repo *repositories.ScheduleRepository) *ScheduleService {
	return &ScheduleService{db: db, repo: repo}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func validateShiftDate(date string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return validationErr("neispravan datum (format YYYY-MM-DD)")
	}
	return nil
}

func toShiftDTO(s repositories.ShiftRow, assignments []repositories.ShiftAssignmentRow) dto.ShiftItem {
	item := dto.ShiftItem{
		ID:          s.ID,
		ProjectID:   s.ProjectID,
		ProjectName: s.ProjectName,
		ShiftDate:   s.ShiftDate,
		StartTime:   s.StartTime,
		EndTime:     s.EndTime,
		Notes:       s.Notes,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
		Assignments: make([]dto.ShiftAssignmentItem, 0, len(assignments)),
	}
	if s.CancelledAt != nil {
		t := s.CancelledAt.Format(time.RFC3339)
		item.CancelledAt = &t
	}
	for _, a := range assignments {
		item.Assignments = append(item.Assignments, dto.ShiftAssignmentItem{
			ID:                a.ID,
			EmployeeID:        a.EmployeeID,
			EmployeeName:      a.EmployeeName,
			OverlapOverridden: a.OverlapOverridden,
		})
	}
	return item
}

func dedupeStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ── Public API ────────────────────────────────────────────────────────────────

func (s *ScheduleService) CreateShift(
	ctx context.Context,
	companyID, projectID, userID string,
	req *dto.CreateShiftRequest,
) (*dto.ShiftItem, error) {
	if err := validateShiftDate(req.ShiftDate); err != nil {
		return nil, err
	}

	ok, err := s.repo.ProjectBelongsToCompany(ctx, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check project: %w", err)
	}
	if !ok {
		return nil, ErrNotFound
	}

	empIDs := dedupeStrings(req.EmployeeIDs)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	shiftID, err := s.repo.CreateShift(ctx, tx, companyID, projectID,
		req.ShiftDate, req.StartTime, req.EndTime, req.Notes, userID)
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("create shift: %w", err)
	}

	for _, empID := range empIDs {
		if err := s.repo.CreateAssignment(ctx, tx, companyID, shiftID, empID, userID,
			false, nil); err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("create assignment for employee %s: %w", empID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.GetShift(ctx, companyID, shiftID)
}

func (s *ScheduleService) GetShift(ctx context.Context, companyID, shiftID string) (*dto.ShiftItem, error) {
	shift, err := s.repo.GetShift(ctx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get shift: %w", err)
	}

	assignments, err := s.repo.ListAssignments(ctx, companyID, shiftID)
	if err != nil {
		return nil, fmt.Errorf("load assignments: %w", err)
	}

	item := toShiftDTO(*shift, assignments)
	return &item, nil
}

func (s *ScheduleService) ListShifts(
	ctx context.Context,
	companyID, dateFrom, dateTo string,
	projectID *string,
) ([]dto.ShiftItem, error) {
	if _, err := time.Parse("2006-01-02", dateFrom); err != nil {
		return nil, validationErr("neispravan datum_od (format YYYY-MM-DD)")
	}
	if _, err := time.Parse("2006-01-02", dateTo); err != nil {
		return nil, validationErr("neispravan datum_do (format YYYY-MM-DD)")
	}

	shifts, err := s.repo.ListShifts(ctx, companyID, dateFrom, dateTo, projectID)
	if err != nil {
		return nil, fmt.Errorf("list shifts: %w", err)
	}

	items := make([]dto.ShiftItem, 0, len(shifts))
	for _, shift := range shifts {
		assignments, err := s.repo.ListAssignments(ctx, companyID, shift.ID)
		if err != nil {
			return nil, fmt.Errorf("load assignments for shift %s: %w", shift.ID, err)
		}
		items = append(items, toShiftDTO(shift, assignments))
	}
	return items, nil
}

func (s *ScheduleService) UpdateShift(
	ctx context.Context,
	companyID, shiftID string,
	req *dto.UpdateShiftRequest,
) (*dto.ShiftItem, error) {
	current, err := s.repo.GetShift(ctx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get shift: %w", err)
	}
	if current.Status == "cancelled" {
		return nil, ErrShiftCancelled
	}

	newDate := current.ShiftDate
	if req.ShiftDate != nil {
		newDate = *req.ShiftDate
	}
	newStart := current.StartTime // *string
	if req.StartTime != nil {
		newStart = req.StartTime
	}
	newEnd := current.EndTime // *string
	if req.EndTime != nil {
		newEnd = req.EndTime
	}
	newNotes := current.Notes
	if req.Notes != nil {
		newNotes = req.Notes
	}

	if err := validateShiftDate(newDate); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	locked, err := s.repo.GetShiftForUpdate(ctx, tx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		tx.Rollback(ctx)
		return nil, ErrNotFound
	}
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("lock shift: %w", err)
	}
	if locked.Status == "cancelled" {
		tx.Rollback(ctx)
		return nil, ErrShiftCancelled
	}

	if err := s.repo.UpdateShift(ctx, tx, shiftID, newDate, newStart, newEnd, newNotes); err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("update shift: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.GetShift(ctx, companyID, shiftID)
}

func (s *ScheduleService) CancelShift(ctx context.Context, companyID, shiftID, userID string) error {
	current, err := s.repo.GetShift(ctx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get shift: %w", err)
	}
	if current.Status == "cancelled" {
		return ErrShiftCancelled
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	locked, err := s.repo.GetShiftForUpdate(ctx, tx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		tx.Rollback(ctx)
		return ErrNotFound
	}
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("lock shift: %w", err)
	}
	if locked.Status == "cancelled" {
		tx.Rollback(ctx)
		return ErrShiftCancelled
	}

	if err := s.repo.CancelShift(ctx, tx, companyID, shiftID, userID); err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("cancel shift: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (s *ScheduleService) SyncAssignments(
	ctx context.Context,
	companyID, shiftID, userID string,
	req *dto.AssignEmployeesRequest,
) (*dto.AssignResponse, error) {
	shift, err := s.repo.GetShift(ctx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get shift: %w", err)
	}
	if shift.Status == "cancelled" {
		return nil, ErrShiftCancelled
	}

	empIDs := dedupeStrings(req.EmployeeIDs)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	locked, err := s.repo.GetShiftForUpdate(ctx, tx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		tx.Rollback(ctx)
		return nil, ErrNotFound
	}
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("lock shift: %w", err)
	}
	if locked.Status == "cancelled" {
		tx.Rollback(ctx)
		return nil, ErrShiftCancelled
	}

	if err := s.repo.DeleteAssignments(ctx, tx, shiftID); err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("delete assignments: %w", err)
	}

	for _, empID := range empIDs {
		if err := s.repo.CreateAssignment(ctx, tx, companyID, shiftID, empID, userID,
			false, nil); err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("create assignment for employee %s: %w", empID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	assignments, err := s.repo.ListAssignments(ctx, companyID, shiftID)
	if err != nil {
		return nil, fmt.Errorf("reload assignments: %w", err)
	}

	result := &dto.AssignResponse{
		Assignments:      make([]dto.ShiftAssignmentItem, 0, len(assignments)),
		RequiresOverride: false,
	}
	for _, a := range assignments {
		result.Assignments = append(result.Assignments, dto.ShiftAssignmentItem{
			ID:                a.ID,
			EmployeeID:        a.EmployeeID,
			EmployeeName:      a.EmployeeName,
			OverlapOverridden: a.OverlapOverridden,
		})
	}
	return result, nil
}

func (s *ScheduleService) DuplicateShift(
	ctx context.Context,
	companyID, shiftID, userID string,
	req *dto.DuplicateShiftRequest,
) (*dto.ShiftItem, error) {
	if _, err := time.Parse("2006-01-02", req.TargetDate); err != nil {
		return nil, validationErr("neispravan ciljni datum (format YYYY-MM-DD)")
	}

	source, err := s.repo.GetShift(ctx, companyID, shiftID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get source shift: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	newID, err := s.repo.CreateShift(ctx, tx, companyID, source.ProjectID,
		req.TargetDate, source.StartTime, source.EndTime, source.Notes, userID)
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("create duplicate shift: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.GetShift(ctx, companyID, newID)
}

func (s *ScheduleService) CopyDay(
	ctx context.Context,
	companyID, userID string,
	req *dto.CopyDayRequest,
) (*dto.CopyDayResponse, error) {
	if _, err := time.Parse("2006-01-02", req.SourceDate); err != nil {
		return nil, validationErr("neispravan izvorni datum (format YYYY-MM-DD)")
	}
	if _, err := time.Parse("2006-01-02", req.TargetDate); err != nil {
		return nil, validationErr("neispravan ciljni datum (format YYYY-MM-DD)")
	}
	if req.SourceDate == req.TargetDate {
		return nil, validationErr("izvorni i ciljni datum moraju biti različiti")
	}

	sourceShifts, err := s.repo.ListShifts(ctx, companyID, req.SourceDate, req.SourceDate, nil)
	if err != nil {
		return nil, fmt.Errorf("list source shifts: %w", err)
	}

	resp := &dto.CopyDayResponse{Shifts: make([]dto.ShiftItem, 0, len(sourceShifts))}

	for _, src := range sourceShifts {
		if src.Status == "cancelled" {
			resp.Skipped++
			continue
		}

		item, err := s.copyShiftToDate(ctx, companyID, userID, src, req.TargetDate)
		if err != nil {
			return nil, err
		}
		resp.Shifts = append(resp.Shifts, *item)
		resp.Created++
	}

	return resp, nil
}

func (s *ScheduleService) CopyWeek(
	ctx context.Context,
	companyID, userID string,
	req *dto.CopyWeekRequest,
) (*dto.CopyDayResponse, error) {
	srcStart, err := time.Parse("2006-01-02", req.SourceWeekStart)
	if err != nil {
		return nil, validationErr("neispravan izvorni tjedan (format YYYY-MM-DD)")
	}
	tgtStart, err := time.Parse("2006-01-02", req.TargetWeekStart)
	if err != nil {
		return nil, validationErr("neispravan ciljni tjedan (format YYYY-MM-DD)")
	}
	if req.SourceWeekStart == req.TargetWeekStart {
		return nil, validationErr("izvorni i ciljni tjedan moraju biti različiti")
	}

	srcEnd := srcStart.AddDate(0, 0, 6).Format("2006-01-02")
	sourceShifts, err := s.repo.ListShifts(ctx, companyID, req.SourceWeekStart, srcEnd, nil)
	if err != nil {
		return nil, fmt.Errorf("list source shifts: %w", err)
	}

	resp := &dto.CopyDayResponse{Shifts: make([]dto.ShiftItem, 0, len(sourceShifts))}

	for _, src := range sourceShifts {
		if src.Status == "cancelled" {
			resp.Skipped++
			continue
		}

		shiftDay, err := time.Parse("2006-01-02", src.ShiftDate)
		if err != nil {
			return nil, fmt.Errorf("parse shift date: %w", err)
		}
		daysOffset := int(shiftDay.Sub(srcStart).Hours() / 24)
		targetDate := tgtStart.AddDate(0, 0, daysOffset).Format("2006-01-02")

		item, err := s.copyShiftToDate(ctx, companyID, userID, src, targetDate)
		if err != nil {
			return nil, err
		}
		resp.Shifts = append(resp.Shifts, *item)
		resp.Created++
	}

	return resp, nil
}

// copyShiftToDate creates a copy of src on targetDate, including all assignments.
func (s *ScheduleService) copyShiftToDate(
	ctx context.Context,
	companyID, userID string,
	src repositories.ShiftRow,
	targetDate string,
) (*dto.ShiftItem, error) {
	srcAssignments, err := s.repo.ListAssignments(ctx, companyID, src.ID)
	if err != nil {
		return nil, fmt.Errorf("load source assignments: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	newID, err := s.repo.CreateShift(ctx, tx, companyID, src.ProjectID,
		targetDate, src.StartTime, src.EndTime, src.Notes, userID)
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("copy shift: %w", err)
	}

	for _, a := range srcAssignments {
		if err := s.repo.CreateAssignment(ctx, tx, companyID, newID,
			a.EmployeeID, userID, false, nil); err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("copy assignment: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.GetShift(ctx, companyID, newID)
}

// EmployeesForDate returns all active company employees annotated with whether
// they already have an active shift on shiftDate. excludeShiftID identifies the
// shift being edited so its own assignments are not counted as conflicts.
func (s *ScheduleService) EmployeesForDate(
	ctx context.Context,
	companyID, shiftDate, excludeShiftID string,
) ([]dto.EmployeeForDateItem, error) {
	if err := validateShiftDate(shiftDate); err != nil {
		return nil, err
	}

	rows, err := s.repo.EmployeesForDate(ctx, companyID, shiftDate, excludeShiftID)
	if err != nil {
		return nil, fmt.Errorf("employees for date: %w", err)
	}

	items := make([]dto.EmployeeForDateItem, len(rows))
	for i, r := range rows {
		items[i] = dto.EmployeeForDateItem{
			ID:          r.ID,
			Name:        r.Name,
			Role:        r.Role,
			Assigned:    r.Assigned,
			ShiftID:     r.ShiftID,
			ProjectName: r.ProjectName,
		}
	}
	return items, nil
}
