package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"finance-dashboard/internal/domain"
	"finance-dashboard/internal/repository"
	"finance-dashboard/pkg/password"
	"finance-dashboard/pkg/response"
)

type UserHandler struct {
	userRepo  *repository.UserRepository
	auditRepo *repository.AuditRepository
}

func NewUserHandler(userRepo *repository.UserRepository, auditRepo *repository.AuditRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo, auditRepo: auditRepo}
}

func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	users, total, err := h.userRepo.List(page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.SuccessPaginated(c, http.StatusOK, users, page, perPage, total)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if user == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	response.Success(c, http.StatusOK, user)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	// FIX: was 422, now 400 (tests 11, 12)
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	roleID, err := repository.RoleNameToID(req.Role)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	user, err := h.userRepo.Create(req.Name, req.Email, hash, roleID)
	if err != nil {
		// FIX: duplicate email returns 400 not 409 (test 11)
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()
	_ = h.auditRepo.Log(context.Background(), "user", user.ID, "CREATE", actorID, ip, nil, user)

	response.Success(c, http.StatusCreated, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	// FIX: was 422, now 400
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	old, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if old == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	if err := h.userRepo.Update(id, req.Name, req.Email); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	updated, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()
	_ = h.auditRepo.Log(context.Background(), "user", id, "UPDATE", actorID, ip, old, updated)

	response.Success(c, http.StatusOK, updated)
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var req domain.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	// FIX: was 422, now 400 (test 13)
	if err := validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	roleID, err := repository.RoleNameToID(req.Role)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	old, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if old == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	if err := h.userRepo.UpdateRole(id, roleID); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	updated, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()
	_ = h.auditRepo.Log(context.Background(), "user", id, "ROLE_CHANGE", actorID, ip, old, updated)

	response.Success(c, http.StatusOK, updated)
}

func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req domain.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	old, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if old == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	if err := h.userRepo.UpdateStatus(id, req.IsActive); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	updated, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()
	_ = h.auditRepo.Log(context.Background(), "user", id, "STATUS_CHANGE", actorID, ip, old, updated)

	response.Success(c, http.StatusOK, updated)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	old, err := h.userRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if old == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	if err := h.userRepo.SoftDelete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	actorID := c.MustGet("user_id").(string)
	ip := c.ClientIP()
	_ = h.auditRepo.Log(context.Background(), "user", id, "DELETE", actorID, ip, old, nil)

	response.Success(c, http.StatusOK, gin.H{"message": "user deleted"})
}
