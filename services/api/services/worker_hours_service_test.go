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

type mockWorkerHoursRepo struct {
	listAllProjectsFn  func(ctx context.Context, companyID string) ([]dto.WorkerProject, error)
	getProjectStatusFn func(ctx context.Context, companyID, projectID string) (string, error)
	otherHoursFn       func(ctx context.Context, companyID, workerEmpID, projectID, workDate string) (float64, error)
	upsertFn           func(ctx context.Context, companyID, workerEmpID, projectID, workDate string, hours float64, notes, workDescription *string, submittedBy string, clientSubmissionID *string) (*dto.WorkerHoursEntry, error)
	listForDateFn      func(ctx context.Context, companyID, workerEmpID, workDate string) ([]dto.WorkerHoursEntry, error)
	listBeforeDateFn   func(ctx context.Context, companyID, workerEmpID, beforeDate string) ([]dto.WorkerHoursEntry, error)
	listForManagerFn   func(ctx context.Context, companyID, projectID, workDate string) ([]dto.ManagerWorkerHoursEntry, error)
	getByIDFn          func(ctx context.Context, companyID, entryID string) (*dto.WorkerHoursDetail, error)
	correctHoursFn     func(ctx context.Context, companyID, entryID string, newHours float64, reason, changedBy string) error
	addCommentFn       func(ctx context.Context, companyID, entryID, authorUserID, text string) (*dto.WorkerHoursComment, error)
}

func (m *mockWorkerHoursRepo) ListCompanyActiveProjects(ctx context.Context, companyID string) ([]dto.WorkerProject, error) {
	if m.listAllProjectsFn != nil {
		return m.listAllProjectsFn(ctx, companyID)
	}
	return []dto.WorkerProject{{ID: "p1", Name: "Projekt A"}, {ID: "p2", Name: "Projekt B"}}, nil
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

func (m *mockWorkerHoursRepo) Upsert(ctx context.Context, companyID, workerEmpID, projectID, workDate string, hours float64, notes, workDescription *string, submittedBy string, clientSubmissionID *string) (*dto.WorkerHoursEntry, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, companyID, workerEmpID, projectID, workDate, hours, notes, workDescription, submittedBy, clientSubmissionID)
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

func (m *mockWorkerHoursRepo) ListForManager(ctx context.Context, companyID, projectID, workDate string) ([]dto.ManagerWorkerHoursEntry, error) {
	if m.listForManagerFn != nil {
		return m.listForManagerFn(ctx, companyID, projectID, workDate)
	}
	return []dto.ManagerWorkerHoursEntry{}, nil
}

func (m *mockWorkerHoursRepo) GetByID(ctx context.Context, companyID, entryID string) (*dto.WorkerHoursDetail, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, companyID, entryID)
	}
	return &dto.WorkerHoursDetail{}, nil
}

func (m *mockWorkerHoursRepo) CorrectHours(ctx context.Context, companyID, entryID string, newHours float64, reason, changedBy string) error {
	if m.correctHoursFn != nil {
		return m.correctHoursFn(ctx, companyID, entryID, newHours, reason, changedBy)
	}
	return nil
}

func (m *mockWorkerHoursRepo) AddComment(ctx context.Context, companyID, entryID, authorUserID, text string) (*dto.WorkerHoursComment, error) {
	if m.addCommentFn != nil {
		return m.addCommentFn(ctx, companyID, entryID, authorUserID, text)
	}
	return &dto.WorkerHoursComment{ID: "comment-id", CommentText: text}, nil
}

func newWHSvc(repo *mockWorkerHoursRepo) *WorkerHoursService {
	return &WorkerHoursService{repo: repo}
}

// ── ListCompanyProjects ───────────────────────────────────────────────────────

func TestListProjects_RadnikSeesAllCompanyProjects(t *testing.T) {
	svc := newWHSvc(&mockWorkerHoursRepo{})
	projects, err := svc.ListCompanyProjects(context.Background(), "comp", "radnik")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestListProjects_PoslovodaSeesAllCompanyProjects(t *testing.T) {
	svc := newWHSvc(&mockWorkerHoursRepo{})
	projects, err := svc.ListCompanyProjects(context.Background(), "comp", "poslovoda")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestListProjects_BothRolesCallSameRepoMethod(t *testing.T) {
	for _, role := range []string{"radnik", "poslovoda"} {
		calls := 0
		repo := &mockWorkerHoursRepo{
			listAllProjectsFn: func(_ context.Context, companyID string) ([]dto.WorkerProject, error) {
				calls++
				if companyID != "comp" {
					t.Errorf("role=%s: unexpected companyID %q", role, companyID)
				}
				return []dto.WorkerProject{}, nil
			},
		}
		svc := newWHSvc(repo)
		_, err := svc.ListCompanyProjects(context.Background(), "comp", role)
		if err != nil {
			t.Errorf("role=%s: unexpected error: %v", role, err)
		}
		if calls != 1 {
			t.Errorf("role=%s: ListCompanyActiveProjects called %d times, want 1", role, calls)
		}
	}
}

func TestListProjects_UnknownRoleForbidden(t *testing.T) {
	for _, role := range []string{"direktor", "inzenjer", "administracija", ""} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		_, err := svc.ListCompanyProjects(context.Background(), "comp", role)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("role=%q: expected ErrForbidden, got %v", role, err)
		}
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

func TestSubmit_RadnikCanSubmitToAnyActiveProject(t *testing.T) {
	upsertCalled := false
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, companyID, _, _, _ string, _ float64, _ *string, _ *string, _ string, _ *string) (*dto.WorkerHoursEntry, error) {
			upsertCalled = true
			if companyID != "comp" {
				t.Errorf("unexpected companyID %q", companyID)
			}
			return &dto.WorkerHoursEntry{ID: "e1"}, nil
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", baseReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upsertCalled {
		t.Error("Upsert was not called")
	}
}

func TestSubmit_PoslovodaCanSubmitToAnyActiveProject(t *testing.T) {
	upsertCalled := false
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, _ *string, _ string, _ *string) (*dto.WorkerHoursEntry, error) {
			upsertCalled = true
			return &dto.WorkerHoursEntry{ID: "e1"}, nil
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "poslovoda", baseReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upsertCalled {
		t.Error("Upsert was not called")
	}
}

func TestSubmit_WorkDescriptionForwardedToRepo(t *testing.T) {
	desc := "Betoniranje temelja"
	var capturedDesc *string
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, wd *string, _ string, _ *string) (*dto.WorkerHoursEntry, error) {
			capturedDesc = wd
			return &dto.WorkerHoursEntry{ID: "e1"}, nil
		},
	}
	req := baseReq()
	req.WorkDescription = &desc
	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "poslovoda", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedDesc == nil || *capturedDesc != desc {
		t.Errorf("expected work_description %q forwarded to repo, got %v", desc, capturedDesc)
	}
}

func TestSubmit_ProjectFromOtherCompanyRejected(t *testing.T) {
	repo := &mockWorkerHoursRepo{
		getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "", nil
		},
	}
	for _, role := range []string{"radnik", "poslovoda"} {
		svc := newWHSvc(repo)
		_, err := svc.Submit(context.Background(), "comp", "emp", "user", role, baseReq())
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("role=%s: expected ValidationError for foreign project, got %v", role, err)
		}
	}
}

func TestSubmit_InactiveProjectRejected(t *testing.T) {
	for _, status := range []string{"archived", "completed", "draft"} {
		repo := &mockWorkerHoursRepo{
			getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) {
				return status, nil
			},
		}
		for _, role := range []string{"radnik", "poslovoda"} {
			svc := newWHSvc(repo)
			_, err := svc.Submit(context.Background(), "comp", "emp", "user", role, baseReq())
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("status=%s role=%s: expected ValidationError, got %v", status, role, err)
			}
		}
	}
}

func TestSubmit_WrongDateRejected(t *testing.T) {
	for _, date := range []string{"2099-01-01", "2020-01-01"} {
		req := baseReq()
		req.WorkDate = date
		for _, role := range []string{"radnik", "poslovoda"} {
			svc := newWHSvc(&mockWorkerHoursRepo{})
			_, err := svc.Submit(context.Background(), "comp", "emp", "user", role, req)
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("date=%s role=%s: expected ValidationError, got %v", date, role, err)
			}
		}
	}
}

func TestSubmit_InvalidHoursRejected(t *testing.T) {
	for _, h := range []float64{-1, 25, 100} {
		req := baseReq()
		req.HoursWorked = h
		for _, role := range []string{"radnik", "poslovoda"} {
			svc := newWHSvc(&mockWorkerHoursRepo{})
			_, err := svc.Submit(context.Background(), "comp", "emp", "user", role, req)
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("hours=%.0f role=%s: expected ValidationError, got %v", h, role, err)
			}
		}
	}
}

func TestSubmit_DailyCapExceeded(t *testing.T) {
	repo := &mockWorkerHoursRepo{
		otherHoursFn: func(_ context.Context, _, _, _, _ string) (float64, error) {
			return 20, nil
		},
	}
	req := baseReq()
	req.HoursWorked = 5
	for _, role := range []string{"radnik", "poslovoda"} {
		svc := newWHSvc(repo)
		_, err := svc.Submit(context.Background(), "comp", "emp", "user", role, req)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("role=%s: expected ValidationError for daily cap, got %v", role, err)
		}
	}
}

func TestSubmit_UnknownRoleForbidden(t *testing.T) {
	for _, role := range []string{"direktor", "inzenjer", "administracija", ""} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		_, err := svc.Submit(context.Background(), "comp", "emp", "user", role, baseReq())
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("role=%q: expected ErrForbidden, got %v", role, err)
		}
	}
}

// ── client_submission_id ──────────────────────────────────────────────────────

// First submission with a new ID succeeds and passes the ID to the repo.
func TestSubmit_WithSubmissionID_FirstSubmit_PassedToRepo(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	var capturedID *string
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, _ *string, _ string, clientSubmissionID *string) (*dto.WorkerHoursEntry, error) {
			capturedID = clientSubmissionID
			return &dto.WorkerHoursEntry{ID: "e1"}, nil
		},
	}
	req := baseReq()
	req.ClientSubmissionID = &id

	svc := newWHSvc(repo)
	entry, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if capturedID == nil || *capturedID != id {
		t.Errorf("submission ID not passed to repo: got %v, want %q", capturedID, id)
	}
}

// When the repo detects that the submission ID was already stored (same payload),
// it returns the existing entry. The service returns it unchanged (200, not 201).
func TestSubmit_WithSubmissionID_Retry_ReturnsOriginalEntry(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	existing := &dto.WorkerHoursEntry{ID: "original-entry-id", HoursWorked: 8}
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, _ *string, _ string, _ *string) (*dto.WorkerHoursEntry, error) {
			return existing, nil
		},
	}
	req := baseReq()
	req.ClientSubmissionID = &id

	svc := newWHSvc(repo)
	entry, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil || entry.ID != "original-entry-id" {
		t.Errorf("expected original entry, got %+v", entry)
	}
}

// When the repo returns ErrSubmissionConflict (same ID, different payload),
// the service maps it to ErrSubmissionConflict (handler will return 409).
func TestSubmit_WithSubmissionID_PayloadMismatch_ReturnsConflict(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, _ *string, _ string, _ *string) (*dto.WorkerHoursEntry, error) {
			return nil, repositories.ErrSubmissionConflict
		},
	}
	req := baseReq()
	req.ClientSubmissionID = &id

	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", req)
	if !errors.Is(err, ErrSubmissionConflict) {
		t.Errorf("expected ErrSubmissionConflict, got %v", err)
	}
}

// Without a submission ID the service passes nil to the repo (legacy path).
func TestSubmit_WithoutSubmissionID_PassesNilToRepo(t *testing.T) {
	var capturedID *string
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, _ *string, _ string, clientSubmissionID *string) (*dto.WorkerHoursEntry, error) {
			capturedID = clientSubmissionID
			return &dto.WorkerHoursEntry{ID: "e1"}, nil
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", baseReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != nil {
		t.Errorf("expected nil submission ID, got %q", *capturedID)
	}
}

// Authorization is checked before the repo is called, even when a submission ID is present.
func TestSubmit_ForbiddenRole_SubmissionIDIgnored(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	upsertCalled := false
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, _ *string, _ string, _ *string) (*dto.WorkerHoursEntry, error) {
			upsertCalled = true
			return &dto.WorkerHoursEntry{}, nil
		},
	}
	req := baseReq()
	req.ClientSubmissionID = &id

	svc := newWHSvc(repo)
	_, err := svc.Submit(context.Background(), "comp", "emp", "user", "direktor", req)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
	if upsertCalled {
		t.Error("Upsert must not be called for forbidden role")
	}
}

func TestSubmit_ConcurrentRetries_BothSucceed(t *testing.T) {
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	calls := 0
	repo := &mockWorkerHoursRepo{
		upsertFn: func(_ context.Context, _, _, _, _ string, _ float64, _ *string, _ *string, _ string, _ *string) (*dto.WorkerHoursEntry, error) {
			calls++
			return &dto.WorkerHoursEntry{ID: "e1"}, nil
		},
	}
	req := baseReq()
	req.ClientSubmissionID = &id
	svc := newWHSvc(repo)

	for i := 0; i < 2; i++ {
		_, err := svc.Submit(context.Background(), "comp", "emp", "user", "radnik", req)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i+1, err)
		}
	}
	if calls != 2 {
		t.Errorf("expected Upsert called twice, got %d", calls)
	}
}

// ── Manager access control ────────────────────────────────────────────────────

func TestGetDetail_ManagerRolesAllowed(t *testing.T) {
	for _, role := range []string{"direktor", "inzenjer"} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		_, err := svc.GetDetail(context.Background(), "comp", role, "entry-id")
		if errors.Is(err, ErrForbidden) {
			t.Errorf("role=%s: expected access, got ErrForbidden", role)
		}
	}
}

func TestGetDetail_NonManagerRolesForbidden(t *testing.T) {
	for _, role := range []string{"radnik", "poslovoda", "administracija", ""} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		_, err := svc.GetDetail(context.Background(), "comp", role, "entry-id")
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("role=%q: expected ErrForbidden, got %v", role, err)
		}
	}
}

func TestGetDetail_NotFoundPropagated(t *testing.T) {
	repo := &mockWorkerHoursRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.WorkerHoursDetail, error) {
			return nil, repositories.ErrWorkerHoursNotFound
		},
	}
	svc := newWHSvc(repo)
	_, err := svc.GetDetail(context.Background(), "comp", "direktor", "missing-id")
	if !errors.Is(err, repositories.ErrWorkerHoursNotFound) {
		t.Errorf("expected ErrWorkerHoursNotFound, got %v", err)
	}
}

func TestCorrectEntry_NonManagerForbidden(t *testing.T) {
	for _, role := range []string{"radnik", "poslovoda", ""} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		err := svc.CorrectEntry(context.Background(), "comp", role, "user", "entry-id",
			dto.CorrectWorkerHoursRequest{NewHours: 8, Reason: "Test"})
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("role=%q: expected ErrForbidden, got %v", role, err)
		}
	}
}

func TestCorrectEntry_InvalidHoursRejected(t *testing.T) {
	for _, h := range []float64{-1, 25} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		err := svc.CorrectEntry(context.Background(), "comp", "direktor", "user", "entry-id",
			dto.CorrectWorkerHoursRequest{NewHours: h, Reason: "Test"})
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("hours=%.0f: expected ValidationError, got %v", h, err)
		}
	}
}

func TestCorrectEntry_ForwardsToRepo(t *testing.T) {
	correctCalled := false
	repo := &mockWorkerHoursRepo{
		correctHoursFn: func(_ context.Context, companyID, entryID string, newHours float64, reason, changedBy string) error {
			correctCalled = true
			if companyID != "comp" || entryID != "entry-id" || newHours != 6 || changedBy != "user-id" {
				return errors.New("unexpected args")
			}
			return nil
		},
	}
	svc := newWHSvc(repo)
	err := svc.CorrectEntry(context.Background(), "comp", "direktor", "user-id", "entry-id",
		dto.CorrectWorkerHoursRequest{NewHours: 6, Reason: "Ispravljeno"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !correctCalled {
		t.Error("CorrectHours was not called on repo")
	}
}

func TestAddComment_NonManagerForbidden(t *testing.T) {
	for _, role := range []string{"radnik", "poslovoda", ""} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		_, err := svc.AddComment(context.Background(), "comp", role, "user", "entry-id", "tekst")
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("role=%q: expected ErrForbidden, got %v", role, err)
		}
	}
}

func TestListManagerEntries_NonManagerForbidden(t *testing.T) {
	for _, role := range []string{"radnik", "poslovoda", ""} {
		svc := newWHSvc(&mockWorkerHoursRepo{})
		_, err := svc.ListManagerEntries(context.Background(), "comp", role, "", "")
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("role=%q: expected ErrForbidden, got %v", role, err)
		}
	}
}
