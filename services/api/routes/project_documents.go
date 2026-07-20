package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

// RegisterProjectDocumentRoutes attaches document and folder routes to the /projects group.
// projects is the *gin.RouterGroup returned by RegisterProjectRoutes.
func RegisterProjectDocumentRoutes(
	projects *gin.RouterGroup,
	h *handlers.ProjectDocumentsHandler,
	folderHandler *handlers.FolderUploadHandler,
	requireRoles func(...string) gin.HandlerFunc,
) {
	manageRoles := requireRoles("direktor", "inzenjer")
	viewRoles := requireRoles("direktor", "inzenjer", "poslovoda", "radnik")

	projects.GET("/:id/documents", viewRoles, h.List)
	projects.POST("/:id/documents", manageRoles, h.Upload)
	projects.GET("/:id/documents/:docId/download", viewRoles, h.Download)
	projects.DELETE("/:id/documents/:docId", manageRoles, h.Delete)

	projects.GET("/:id/folders", viewRoles, folderHandler.ListFolders)
}
