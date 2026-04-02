package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/service"
	"finance-dashboard/pkg/response"
)

type CategoryHandler struct {
	dashSvc *service.DashboardService
}

func NewCategoryHandler(dashSvc *service.DashboardService) *CategoryHandler {
	return &CategoryHandler{dashSvc: dashSvc}
}

func (h *CategoryHandler) List(c *gin.Context) {
	cats, err := h.dashSvc.GetCategories()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusOK, cats)
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req domain.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := validate.Struct(req); err != nil {
		response.ValidationError(c, err)
		return
	}

	cat, err := h.dashSvc.CreateCategory(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, cat)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "invalid category id")
		return
	}

	var req domain.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := validate.Struct(req); err != nil {
		response.ValidationError(c, err)
		return
	}

	updated, err := h.dashSvc.UpdateCategory(id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if updated == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "category not found")
		return
	}

	response.Success(c, http.StatusOK, updated)
}
