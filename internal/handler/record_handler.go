package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/service"
	"finance-dashboard/pkg/response"
)

type RecordHandler struct {
	recordSvc *service.RecordService
}

func NewRecordHandler(recordSvc *service.RecordService) *RecordHandler {
	return &RecordHandler{recordSvc: recordSvc}
}

func (h *RecordHandler) List(c *gin.Context) {
	categoryID, _ := strconv.Atoi(c.Query("category_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	filter := domain.RecordFilter{
		Type:       c.Query("type"),
		Status:     c.Query("status"),
		DateFrom:   c.Query("date_from"),
		DateTo:     c.Query("date_to"),
		Sort:       c.Query("sort"),
		CategoryID: categoryID,
		Page:       page,
		PerPage:    perPage,
	}

	records, total, err := h.recordSvc.List(filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.SuccessPaginated(c, http.StatusOK, records, page, perPage, total)
}

func (h *RecordHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	rec, err := h.recordSvc.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if rec == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "record not found")
		return
	}
	response.Success(c, http.StatusOK, rec)
}

func (h *RecordHandler) Create(c *gin.Context) {
	var req domain.CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	rec, err := h.recordSvc.Create(&req, actorID, ip)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, rec)
}

func (h *RecordHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req domain.UpdateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	rec, err := h.recordSvc.Update(id, &req, actorID, ip)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if rec == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "record not found")
		return
	}
	response.Success(c, http.StatusOK, rec)
}

func (h *RecordHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	if err := h.recordSvc.Delete(id, actorID, ip); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "record deleted"})
}

func (h *RecordHandler) Void(c *gin.Context) {
	id := c.Param("id")
	var req domain.VoidRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()

	if err := h.recordSvc.Void(id, req.Reason, actorID, ip); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "record voided"})
}
