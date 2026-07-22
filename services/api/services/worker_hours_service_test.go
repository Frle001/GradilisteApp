package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gradiliste/api/dto"
)

// ── Mock repository ───────────────────────────────────────────────────────────

type mockWorkerHoursRepo struct {
	listProjectsFn     func(ctx context.Context, companyID, empID, roleOnProject string) ([]dto.WorkerProject, error)
	hasAssignmentFn    func(ctx context.Context, empID, projectID, roleOnProject string) (bool, error)
	getProjectStatusFn func(ctx context.Context, companyID, projectID string) (string, error)
	otherHoursFn       func(ctx context.Context, companyID, workerEmpID, projectID, workDate string) (float64, error)
	upsertFn           func(ctx context.Context, companyID, workerEmpID, projectID, workDate string, hours float64, notes *string, submittedBy string) (*dto.WorkerHoursEntry, error)
	listForDateFn      func(ctx context.Context, companyID, workerEmpID, workDate string) ([]dto.WorkerHoursEntry, error)
	listBeforeDateFn   func(ctx context.Context, companyID, workerEmpID, beforeDate string) ([]dto.WorkerHoursEntry, error)
}

func (m *mockWorkerHoursRepo) ListActiveProjectsByAssignment(ctx context.Context, companyID, empID, roleOnProject string) ([]dto.WorkerProject, error) {
	if m.listProjectsFn != nil {
		return m.listProjectsFn(ctx, companyID, empID, roleOnProject)
	}
	return []dto.WorkerProject{}, nil
}

func (m *mockWorkerHoursRepo) HasActiveAssignment(ctx context.Context, empID, projectID, roleOnProject string) (bool, error) {
	if m.hasAssignmentFn != nil {
		return m.hasAssignmentFn(ctx, empID, projectID, roleOnProject)
	}
	return true, nil
}

func (m *mockWorkerHoursRepo) GetProjectStatus(ctx context.Context, companyID, projectID string) (string, error) {
	if m.getProjectStatusFn != nil {
		return m.getProjectStatusFn(ctx, companyID, projectID)
	}
	return "active", nil
}

func (m *mockWorkerHoursRepo) GetOtherProjectsHoursForDate(ctx context.Context, companyID, workerEmpID, projectID, workDate string) (float64, error) {
	if m.otherHoursFn != nil {
		return m.otherHoursFn(ctx, companyID, workerEmpID, projectID, workDate)
	}
	return 0, nil
}

func (m *mockWorkerHoursRepo) Upsert(ctx context.Context, companyID, workerEmpID, projectID, workDate string, hours float64, notes *string, submittedBy string) (*dto.WorkerHoursEntry, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, companyID, workerEmpID, projectID, workDate, hours, notes, submittedBy)
	}
	return &dto.WorkerHoursEntry{ID: "entry-id"}, nil
}

func (m *mockWorkerHoursRepo) ListForWorkerDate(ctx context.Context, companyID, workerEmpID, workDate string) ([]dto.WorkerHoursEntry, error) {
	if m.listForDateFn != nil {
		return m.listForDateFn(ctx, companyID, workerEmpID, workDate)
	}
	return []dto.WorkerHoursEntry{}, nil
}

func (m *mockWorkerHoursRepo) ListBeforeDate(ctx context.Context, companyID, workerEmpID, beforeDate string) ([]dto.WorkerHoursEntry, error) {
	if m.listBeforeDateFn != nil {
		return m.listBeforeDateFn(ctx, companyID, workerEmpID, beforeDate)
	}
	return []dto.WorkerHoursEntry{}, nil
}

func newWHSvc(repo *mockWorkerHoursRepo) *WorkerHoursService {
	return &WorkerHoursService{repo: repo}
}

// ── appRoleToProjectRole ──────────────────────────────────────────────────────

func TestAppRoleToProjectRole(t *testing.T) {
	cases := []struct{ in, want string }{
		{"radnik", "worker"},
		{"poslovoda", "poslovoda"},
		{"direktor", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := appRoleToProjectRole(tc.in)
		if got != tc.want {
			t.Errorf("appRoleToProjectRole(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── ListCompanyProjects ───────────────────────────────────────────────────────

func TestListCompanyProjects_RadnikPassesWorkerRole(t *testing.T) {
	var capturedRole string
	repo := &mockWorkerHoursRepo{
		listProjectsFn: func(_ context.Context, _, _, roleOnProject string) ([]dto.WorkerProject, error) {
			capturedRole = roleOnProject
			return []dto.WorkerProject{{ID: "p1", Name: "Projekt"}}, nil
		},
	}
	svc := newWHSvc(repo)
	projects, err := svc.ListCompanyProjects(context.Background(), "comp", "emp", "radnik")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if capturedRole != "worker" {
		t.Errorf("expected role_on_project='worker', got %q", capturedRole)
	}
}

func TestListCompanyProjects_PoslovodaPassesPoslovodaRole(t *testing.T) {
	var capturedRole string
	repo := &mockWorkerHoursRepo{
		listProjectsFn: func(_ context.Context, _, _, roleOnProject string) ([]dto.WorkerProject, error) {
			capturedRole = roleOnProject
			return []dto.WorkerProject{}, nil
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.ListCompanyProjects(context.Background(), "comp", "emp", "poslovoda")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRole != "poslovoda" {
		t.Errorf("expected role_on_project='poslovoda', got %q", capturedRole)
	}
}

func TestListCompanyProjects_UnknownRoleForbidden(t *testing.T) {
	svc := newWHSvc(&mockWorkerHoursRepo{})
	_, err := svc.ListCompanyProjects(context.Background(), "comp", "emp", "direktor")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ── Submit ────────────────────────────────────────────────────────────────────

func todayZagreb() string {
	loc, _ := time.LoadLocation("Europe/Zagreb")
	return time.Now().In(loc).Format("2006-01-02")
}

func baseReq() dto.SubmitWorkerHoursRequest {
	return dto.SubmitWorkerHoursRequest{
		ProjectID:   "proj-id",
		WorkDate:    todayZagreb(),
		HoursWorked: 8,
	}
}

func TestSubmit_RadnikRequiresWorkerAssignment(t *testing.T) {
	var capturedRole string
	repo := &mockWorkerHoursRepo{
		hasAssignmentFn: func(_ context.Context, _, _, roleOnProject string) (bool, error) {
			capturedRole = roleOnProject
			return true, nil
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", baseReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRole != "worker" {
		t.Errorf("expected assignment check for role='worker', got %q", capturedRole)
	}
}

func TestSubmit_PoslovodaRequiresPoslovodaAssignment(t *testing.T) {
	var capturedRole string
	repo := &mockWorkerHoursRepo{
		hasAssignmentFn: func(_ context.Context, _, _, roleOnProject string) (bool, error) {
			capturedRole = roleOnProject
			return true, nil
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "poslovoda", baseReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRole != "poslovoda" {
		t.Errorf("expected assignment check for role='poslovoda', got %q", capturedRole)
	}
}

func TestSubmit_NoAssignmentReturnsForbidden(t *testing.T) {
	repo := &mockWorkerHoursRepo{
		hasAssignmentFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", baseReq())
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSubmit_UnknownRoleForbidden(t *testing.T) {
	svc := newWHSvc(&mockWorkerHoursRepo{})
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "direktor", baseReq())
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSubmit_FutureDateRejected(t *testing.T) {
	req := baseReq()
	req.WorkDate = "2099-01-01"
	svc := newWHSvc(&mockWorkerHoursRepo{})
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", req)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError for future date, got %v", err)
	}
}

func TestSubmit_HoursExceed24Rejected(t *testing.T) {
	repo := &mockWorkerHoursRepo{
		otherHoursFn: func(_ context.Context, _, _, _, _ string) (float64, error) {
			return 20, nil
		},
	}
	svc := newWHSvc(repo)
	req := baseReq()
	req.HoursWorked = 5 // 20 + 5 > 24
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", req)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError for exceeding 24h, got %v", err)
	}
}
