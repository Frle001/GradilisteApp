package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/appctx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// projectViewRoles mirrors the roles allowed to call GET /api/projects/:id.
// direktor, inzenjer, poslovoda, radnik — administracija is excluded.
var projectViewRoles = []string{"direktor", "inzenjer", "poslovoda", "radnik"}

func testContext(role string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/projects/test-id", nil)
	c.Set(appctx.AuthUserKey, &appctx.AuthContext{
		UserID:    "user-1",
		CompanyID: "company-1",
		Role:      role,
	})
	return c, w
}

func TestProjectDetail_AllowedRoles(t *testing.T) {
	for _, role := range projectViewRoles {
		c, w := testContext(role)
		h := RequireRoles(projectViewRoles...)
		h(c)
		if w.Code == http.StatusForbidden {
			t.Errorf("role %q should be allowed to view project detail, got 403", role)
		}
	}
}

func TestProjectDetail_AdministracijaForbidden(t *testing.T) {
	c, w := testContext("administracija")
	h := RequireRoles(projectViewRoles...)
	h(c)
	if w.Code != http.StatusForbidden {
		t.Errorf("role 'administracija' must not access project detail, expected 403 got %d", w.Code)
	}
}

func TestProjectDetail_UnknownRoleForbidden(t *testing.T) {
	c, w := testContext("unknown-role")
	h := RequireRoles(projectViewRoles...)
	h(c)
	if w.Code != http.StatusForbidden {
		t.Errorf("unknown role must not access project detail, expected 403 got %d", w.Code)
	}
}

// TestProjectDetail_AfterRoleChange_OldTokenRejected documents the expectation
// that after a user's role changes (which triggers session invalidation), the
// old access token is rejected by the auth_version check before reaching the
// role guard. The user must re-authenticate to get a token with the updated role.
func TestProjectDetail_AfterRoleChange_OldTokenRejected(t *testing.T) {
	// Before role change: direktor with auth_version = 1 → allowed.
	if !checkAuthVersionAllowed(1, 1) {
		t.Fatal("precondition: token must be valid before role change")
	}

	// Role change invalidates sessions (auth_version incremented to 2).
	// Old token (auth_version = 1) must be rejected by auth middleware.
	if checkAuthVersionAllowed(1, 2) {
		t.Error("token from before role-change invalidation must be rejected")
	}

	// New login after role change → auth_version = 2 → accepted.
	if !checkAuthVersionAllowed(2, 2) {
		t.Error("token issued after role change must be accepted")
	}
}

// TestProjectManagement_NonManagerForbidden verifies that only direktor and
// inzenjer can mutate projects (POST/PUT on /api/projects).
func TestProjectManagement_NonManagerForbidden(t *testing.T) {
	manageRoles := []string{"direktor", "inzenjer"}
	nonManagers := []string{"poslovoda", "radnik", "administracija"}

	for _, role := range nonManagers {
		c, w := testContext(role)
		h := RequireRoles(manageRoles...)
		h(c)
		if w.Code != http.StatusForbidden {
			t.Errorf("role %q must not manage projects, expected 403 got %d", role, w.Code)
		}
	}
}
