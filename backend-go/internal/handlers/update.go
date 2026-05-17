package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/BenedictKing/ccx/internal/updater"
	"github.com/gin-gonic/gin"
)

func CheckUpdateHandler(u *updater.Updater) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		status, err := u.CheckUpdate(ctx)
		if err != nil {
			if status != nil {
				c.JSON(http.StatusBadGateway, status)
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, status)
	}
}

func ApplyUpdateHandler(u *updater.Updater) gin.HandlerFunc {
	return func(c *gin.Context) {
		if u.IsDocker() {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Docker 环境不支持内置升级，请使用 Watchtower 或拉取新镜像",
			})
			return
		}

		if u.IsUpdating() {
			c.JSON(http.StatusConflict, gin.H{
				"error": "升级正在进行中",
			})
			return
		}

		status := u.GetLastStatus()
		if status == nil || !status.HasUpdate {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "没有可用的更新，请先检查更新",
			})
			return
		}

		if !status.CanUpdate {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": status.UpdateDisabledReason,
			})
			return
		}

		if !u.StartUpdate() {
			c.JSON(http.StatusConflict, gin.H{
				"error": "升级正在进行中",
			})
			return
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := u.ApplyStarted(ctx); err != nil {
				// ApplyStarted 失败时更新器会释放升级锁
				_ = err
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "update started, server will restart shortly",
		})
	}
}
