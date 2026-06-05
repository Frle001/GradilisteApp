package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

// RegisterProjectMaterialRoutes attaches material routes to the /projects group.
// projects is the *gin.RouterGroup returned by RegisterProjectRoutes.
// Import endpoints are restricted to direktor/inzenjer only.
func RegisterProjectMaterialRoutes(
	projects *gin.RouterGroup,
	h *handlers.ProjectMaterialHandler,
	requireRoles func(...string) gin.HandlerFunc,
) {
	manageRoles := requireRoles("direktor", "inzenjer")

	// authRequired is already on the parent group — no need to add it again.
	projects.GET("/:id/materials", h.List)
	projects.POST("/:id/materials", manageRoles, h.Create)
	projects.PUT("/:id/materials/:materialId", manageRoles, h.Update)
	projects.PATCH("/:id/materials/:materialId/deactivate", manageRoles, h.Deactivate)

	// Phase 6.5 wizard import flow (3 steps):
	//   Step 1: POST /import/analyze  — upload xlsx, detect headers, suggest mappings
	//   Step 2: POST /import/preview  — send import_job_id + user mapping, get parsed preview
	//   Step 3: POST /import/confirm  — send import_job_id, upsert valid rows
	projects.POST("/:id/materials/import/analyze", manageRoles, h.ImportAnalyze)
	projects.POST("/:id/materials/import/preview", manageRoles, h.ImportWizardPreview)
	projects.POST("/:id/materials/import/confirm", manageRoles, h.ImportConfirm)
}
