package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Recovery 返回一个 panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return RecoveryWithWriter(nil)
}

// RecoveryWithWriter 返回一个带自定义日志写入器的 panic 恢复中间件
func RecoveryWithWriter(writer gin.RecoveryFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录堆栈跟踪
				stack := debug.Stack()

				// 记录详细的错误信息
				log.WithFields(log.Fields{
					"error":      err,
					"stack":      string(stack),
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
					"client_ip":  c.ClientIP(),
					"user_agent": c.Request.UserAgent(),
					"timestamp":  time.Now().Format(time.RFC3339),
				}).Error("Panic recovered")

				// 如果提供了自定义写入器，调用它
				if writer != nil {
					writer(c, err)
				}

				// 返回 500 错误
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"message": "Internal server error",
						"type":    "internal_error",
						"code":    "panic_recovered",
					},
				})
			}
		}()

		c.Next()
	}
}

// SafeGo 安全地启动 goroutine，带 panic 恢复
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				log.WithFields(log.Fields{
					"error": err,
					"stack": string(stack),
				}).Error("Goroutine panic recovered")
			}
		}()
		fn()
	}()
}

// SafeGoWithContext 安全地启动 goroutine，带上下文和 panic 恢复
func SafeGoWithContext(name string, fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				log.WithFields(log.Fields{
					"goroutine": name,
					"error":     err,
					"stack":     string(stack),
					"timestamp": time.Now().Format(time.RFC3339),
				}).Error("Named goroutine panic recovered")
			}
		}()
		fn()
	}()
}

// RecoverToError 将 panic 转换为 error（用于函数内部）
func RecoverToError() error {
	if r := recover(); r != nil {
		stack := debug.Stack()
		log.WithFields(log.Fields{
			"error": r,
			"stack": string(stack),
		}).Error("Panic recovered and converted to error")

		return fmt.Errorf("panic recovered: %v", r)
	}
	return nil
}

// SafeCall 安全地调用函数，捕获 panic 并返回 error
func SafeCall(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			log.WithFields(log.Fields{
				"error": r,
				"stack": string(stack),
			}).Error("Panic in SafeCall")

			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return fn()
}

// SafeCallWithValue 安全地调用返回值的函数，捕获 panic
func SafeCallWithValue[T any](fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			log.WithFields(log.Fields{
				"error": r,
				"stack": string(stack),
			}).Error("Panic in SafeCallWithValue")

			var zero T
			result = zero
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	return fn()
}
