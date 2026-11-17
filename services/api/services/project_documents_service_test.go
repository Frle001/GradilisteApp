package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── mock ─────────────────────────────────────────────────────────────────────

type mockProjectDocRepo struct {
	belongsFn      func(ctx context.Context, companyID, projectID string) (bool, error)
	listInFolderFn func(ctx context.Context, companyID, projectID string, folderID *string) ([]dto.ProjectDocumentItem, error)
}

func (m *mockProjectDocRepo) ProjectBelongsToCompany(ctx context.Context, companyID, projectID string) (bool, error) {
	if m.belongsFn != nil {
		return m.belongsFn(ctx, companyID, projectID)
	}
	return true, nil
}

func (m *mockProjectDocRepo) ListInFolder(ctx context.Context, companyID, projectID string, folderID *string) ([]dto.ProjectDocumentItem, error) {
	if m.listInFolderFn != nil {
		return m.listInFolderFn(ctx, companyID, projectID, folderID)
	}
	return nil, nil
}

func (m *mockProjectDocRepo) Create(_ context.Context, _, _, _, _, _, _ string, _ int64) (string, error) {
	return "", nil
}

func (m *mockProjectDocRepo) GetByID(_ context.Context, _, _, _ string) (*repositories.ProjectDocumentRecord, error) {
	return nil, repositories.ErrNotFound
}

func (m *mockProjectDocRepo) SoftDelete(_ context.Context, _, _, _ string) error {
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newProjDocSvc(repo docRepo) *ProjectDocumentsService {
	return &ProjectDocumentsService{repo: repo, storage: &mockStorage{}}
}

func strPtr(s string) *string { return &s }

func makeDoc(id string, folderID *string) dto.ProjectDocumentItem {
	return dto.ProjectDocumentItem{
		ID:           id,
		OriginalName: id + ".pdf",
		ContentType:  "application/pdf",
		FileSize:     1024,
		CreatedAt:    time.Now(),
		FolderID:     folderID,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestDocList_RootReturnsOnlyRootDocuments verifies that listing with
// folderID=nil returns only documents at the project root (folder_id IS NULL).
// Documents inside any subfolder must not appear in the root listing.
//
// Regression for: commit 44c1abe replaced the flat List (all docs) with
// ListInFolder (scoped), so callers that pass nil must receive only root-level
// documents — the query filters on `pd.folder_id IS NOT DISTINCT FROM NULL`.
func TestDocList_RootReturnsOnlyRootDocuments(t *testing.T) {
	rootDoc := makeDoc("root-1", nil)
	folderDoc := makeDoc("folder-1", strPtr("folder-uuid"))

	repo := &mockProjectDocRepo{
		listInFolderFn: func(_ context.Context, _, _ string, folderID *string) ([]dto.ProjectDocumentItem, error) {
			if folderID != nil {
				return []dto.ProjectDocumentItem{folderDoc}, nil
			}
			return []dto.ProjectDocumentItem{rootDoc}, nil
		},
	}

	svc := newProjDocSvc(repo)
	ctx := context.Background()

	docs, err := svc.List(ctx, "company-1", "direktor", "emp-1", "project-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 root doc, got %d", len(docs))
	}
	if docs[0].FolderID != nil {
		t.Error("root listing must not include documents with a non-nil folder_id")
	}
	if docs[0].ID != "root-1" {
		t.Errorf("expected document id 'root-1', got %q", docs[0].ID)
	}
}

// TestDocList_FolderScopedReturnsFolderDocuments verifies that listing with
// a specific folderID returns only documents from that folder.
func TestDocList_FolderScopedReturnsFolderDocuments(t *testing.T) {
	folderID := "folder-uuid"
	folderDoc := makeDoc("folder-1", strPtr(folderID))
	rootDoc := makeDoc("root-1", nil)

	repo := &mockProjectDocRepo{
		listInFolderFn: func(_ context.Context, _, _ string, fid *string) ([]dto.ProjectDocumentItem, error) {
			if fid != nil && *fid == folderID {
				return []dto.ProjectDocumentItem{folderDoc}, nil
			}
			return []dto.ProjectDocumentItem{rootDoc}, nil
		},
	}

	svc := newProjDocSvc(repo)
	ctx := context.Background()

	docs, err := svc.List(ctx, "company-1", "inzenjer", "emp-1", "project-1", &folderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 folder doc, got %d", len(docs))
	}
	if docs[0].ID != "folder-1" {
		t.Errorf("expected document 'folder-1' in folder listing, got %q", docs[0].ID)
	}
}

// TestDocList_UnknownProjectReturnsNotFound confirms that a project not owned
// by the caller's company returns ErrDocumentNotFound (not a 500 or empty list).
func TestDocList_UnknownProjectReturnsNotFound(t *testing.T) {
	repo := &mockProjectDocRepo{
		belongsFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
	}

	svc := newProjDocSvc(repo)
	_, err := svc.List(context.Background(), "company-1", "direktor", "emp-1", "other-project", nil)
	if !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("expected ErrDocumentNotFound, got %v", err)
	}
}

// TestDocList_RepoErrorPropagates verifies that a database error from ListInFolder
// (such as a missing schema object — column folder_id or table project_folders)
// is propagated to the caller rather than silently returning an empty list.
//
// Regression guard: if production is missing migration 016, the backend logs
// a DB error and returns 500; the frontend shows an error state, not an empty
// document list. This test confirms the error path is never swallowed.
func TestDocList_RepoErrorPropagates(t *testing.T) {
	dbErr := errors.New(`column "folder_id" does not exist`)
	repo := &mockProjectDocRepo{
		listInFolderFn: func(_ context.Context, _, _ string, _ *string) ([]dto.ProjectDocumentItem, error) {
			return nil, dbErr
		},
	}

	svc := newProjDocSvc(repo)
	docs, err := svc.List(context.Background(), "company-1", "direktor", "emp-1", "project-1", nil)
	if err == nil {
		t.Error("expected error from repo, got nil")
	}
	if docs != nil {
		t.Error("expected nil docs on repo error, got non-nil")
	}
}

// TestDocList_LegacyDocumentsAtRoot confirms that documents uploaded before
// folder support existed (folder_id = NULL) remain visible at the project root
// after migration 016 is applied.
func TestDocList_LegacyDocumentsAtRoot(t *testing.T) {
	legacyDocs := []dto.ProjectDocumentItem{
		makeDoc("legacy-1", nil),
		makeDoc("legacy-2", nil),
	}

	repo := &mockProjectDocRepo{
		listInFolderFn: func(_ context.Context, _, _ string, folderID *string) ([]dto.ProjectDocumentItem, error) {
			if folderID == nil {
				return legacyDocs, nil
			}
			return nil, nil
		},
	}

	svc := newProjDocSvc(repo)
	docs, err := svc.List(context.Background(), "company-1", "poslovoda", "emp-1", "project-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 legacy docs at root, got %d", len(docs))
	}
	for _, d := range docs {
		if d.FolderID != nil {
			t.Errorf("legacy document %q must have folder_id=nil", d.ID)
		}
	}
}
