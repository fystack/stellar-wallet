package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	workspaces *service.WorkspaceService
}

func (h *WorkspaceHandler) list(c *gin.Context) {
	list, err := h.workspaces.List(c.GetString("userID"), c.GetString("workspaceID"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, list)
}

type createWorkspaceBody struct {
	Name string `json:"name"`
}

func (h *WorkspaceHandler) create(c *gin.Context) {
	var body createWorkspaceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	result, err := h.workspaces.Create(c.GetString("userID"), auth.CookieToken(c.Request), body.Name)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusCreated, result)
}

func (h *WorkspaceHandler) switchActive(c *gin.Context) {
	workspace, err := h.workspaces.Switch(c.GetString("userID"), auth.CookieToken(c.Request), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) members(c *gin.Context) {
	members, err := h.workspaces.Members(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, members)
}

func (h *WorkspaceHandler) removeMember(c *gin.Context) {
	if err := h.workspaces.RemoveMember(c.GetString("userID"), c.Param("id"), c.Param("userId")); err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusNoContent, nil)
}

type inviteBody struct {
	Email string `json:"email"`
}

func (h *WorkspaceHandler) invite(c *gin.Context) {
	var body inviteBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	result, err := h.workspaces.Invite(c.GetString("userID"), c.Param("id"), body.Email)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusCreated, result)
}

func (h *WorkspaceHandler) revokeInvite(c *gin.Context) {
	if err := h.workspaces.RevokeInvite(c.GetString("userID"), c.Param("id"), c.Param("inviteId")); err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusNoContent, nil)
}

func (h *WorkspaceHandler) previewInvite(c *gin.Context) {
	preview, err := h.workspaces.PreviewInvite(c.Param("token"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, preview)
}

func (h *WorkspaceHandler) acceptInvite(c *gin.Context) {
	result, err := h.workspaces.Accept(c.GetString("userID"), auth.CookieToken(c.Request), c.Param("token"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, result)
}

func (h *WorkspaceHandler) invitations(c *gin.Context) {
	invitations, err := h.workspaces.Invitations(c.GetString("userID"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, invitations)
}

func (h *WorkspaceHandler) acceptInvitation(c *gin.Context) {
	result, err := h.workspaces.AcceptInvitation(c.GetString("userID"), auth.CookieToken(c.Request), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, result)
}

func (h *WorkspaceHandler) declineInvitation(c *gin.Context) {
	if err := h.workspaces.DeclineInvitation(c.GetString("userID"), c.Param("id")); err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusNoContent, nil)
}
