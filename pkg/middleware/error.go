package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sukryu/pAuth/pkg/errors"
)

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			log.Printf("Error: %v", err)

			switch e := err.(type) {
			case *errors.StatusError:
				response := gin.H{
					"error": gin.H{
						"code":    e.Code,
						"message": e.Message,
					},
				}
				if e.Reason != "" {
					response["error"].(gin.H)["reason"] = e.Reason
				}
				if e.RetryAfter > 0 {
					response["error"].(gin.H)["retryAfter"] = e.RetryAfter
				}
				c.JSON(e.Code, response)
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    http.StatusInternalServerError,
						"message": "Internal server error",
					},
				})
			}
		}
	}
}
