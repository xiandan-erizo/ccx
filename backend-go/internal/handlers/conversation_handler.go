package handlers

import (
	"net/http"

	"github.com/BenedictKing/ccx/internal/conversation"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/gin-gonic/gin"
)

type ConversationHandlerDeps struct {
	Tracker          *conversation.ConversationTracker
	OverrideManager  *conversation.OverrideManager
	ChannelScheduler *scheduler.ChannelScheduler
}

func GetConversations(deps *ConversationHandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		kindFilter := c.Query("kind")

		conversations := deps.Tracker.GetActiveConversations(kindFilter)
		overrides := deps.OverrideManager.GetAllOverrides()

		overridesResponse := make(map[string]interface{})
		for id, override := range overrides {
			overridesResponse[id] = gin.H{
				"sequence":  override.Sequence,
				"setAt":     override.SetAt,
				"expiresAt": override.ExpiresAt,
			}
		}

		channelsByKind := gin.H{}
		for _, kind := range []scheduler.ChannelKind{
			scheduler.ChannelKindMessages,
			scheduler.ChannelKindChat,
			scheduler.ChannelKindImages,
			scheduler.ChannelKindResponses,
			scheduler.ChannelKindGemini,
		} {
			channelsByKind[string(kind)] = deps.ChannelScheduler.GetConversationChannelsByKind(kind)
		}

		c.JSON(http.StatusOK, gin.H{
			"conversations":  conversations,
			"total":          len(conversations),
			"overrides":      overridesResponse,
			"channelsByKind": channelsByKind,
		})
	}
}

type SetOverrideRequest struct {
	Sequence []conversation.ChannelEntry `json:"sequence" binding:"required,min=1"`
}

func SetConversationOverride(deps *ConversationHandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		convID := c.Param("id")
		if convID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id is required"})
			return
		}

		var req SetOverrideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		conv, ok := deps.Tracker.GetConversation(convID)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
			return
		}

		err := deps.OverrideManager.SetOverride(convID, conv.Kind, conv.RawUserID, req.Sequence)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "override set successfully",
			"conversationId": convID,
			"sequence":       req.Sequence,
		})
	}
}

func RemoveConversationOverride(deps *ConversationHandlerDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		convID := c.Param("id")
		if convID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id is required"})
			return
		}

		removed := deps.OverrideManager.RemoveOverride(convID)
		if !removed {
			c.JSON(http.StatusNotFound, gin.H{"error": "no override found for this conversation"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "override removed",
			"conversationId": convID,
		})
	}
}
