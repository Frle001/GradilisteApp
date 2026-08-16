package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/handlers"
)

func RegisterMaterialPurchasesRoutes(
	api *gin.RouterGroup,
	h *handlers.MaterialPurchasesHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	allRoles := requireRoles("direktor", "inzenjer", "poslovoda")
	creators := requireRoles("direktor", "inzenjer", "poslovoda")
	editors := requireRoles("direktor", "inzenjer")

	rg := api.Group("/material-purchases", authRequired, allRoles)
	{
		// form-data MUST be registered before /:id to avoid Gin treating "form-data" as an :id param
		rg.GET("/form-data", h.FormData)

		rg.GET("", h.List)
		rg.POST("", creators, h.Create)
		rg.GET("/:id", h.GetByID)
		rg.PATCH("/:id", editors, h.Update)
		rg.GET("/:id/receipt", h.GetReceipt)
		rg.DELETE("/:id", h.Delete)
	}
}
