package handler

import (
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/gin-gonic/gin"
)

type DebugHelperHandler struct {
	*Handler
	Registry RegistryHelper
}

type RegistryHelper interface {
	ExportRegistryData() loadbalancer.Registry
}

func NewDebugHelperHandler(h *Handler, r RegistryHelper) *DebugHelperHandler {
	if h == nil {
		panic("handler is nil")
	}
	if r == nil {
		panic("registry helper is nil")
	}
	return &DebugHelperHandler{
		Handler:  h,
		Registry: r,
	}
}

func (h *DebugHelperHandler) RegisterRoutes(r gin.IRouter) {
	group := r.Group("/_debug")
	group.GET("/registry", h.ExportRegistryData)
}

func (h *DebugHelperHandler) ExportRegistryData(c *gin.Context) {
	data := h.Registry.ExportRegistryData()
	c.JSON(200, data)
}
