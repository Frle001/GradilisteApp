package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterProjectRoutes(
	api *gin.RouterGroup,
	h *handlers.ProjectHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) *gin.RouterGroup {
	manageRoles := requireRoles("direktor", "inzenjer")
	archiveRoles := requireRoles("direktor", "inzenjer")
	viewRoles := requireRoles("direktor", "inzenjer", "poslovoda", "radnik")
	direktorOnly := requireRoles("direktor")

	projects := api.Group("/projects", authRequired)
	{
		// Static routes before /:id to prevent "archive" being parsed as an ID
		projects.GET("", viewRoles, h.List)
		projects.GET("/archive", archiveRoles, h.ListArchive)
		projects.POST("", manageRoles, h.Create)

		projects.GET("/:id", viewRoles, h.GetByID)
		projects.PUT("/:id", manageRoles, h.Update)
		projects.DELETE("/:id", direktorOnly, h.Delete)

		projects.PATCH("/:id/assign-poslovoda", manageRoles, h.AssignPoslovoda)
		projects.PATCH("/:id/close", manageRoles, h.Close)
		projects.PATCH("/:id/archive", manageRoles, h.Archive)
		projects.PATCH("/:id/reactivate", manageRoles, h.Reactivate)

		projects.GET("/:id/assignments", viewRoles, h.ListAssignments)
		projects.GET("/:id/archive-summary", archiveRoles, h.GetArchiveSummary)
	}
	return projects
}
