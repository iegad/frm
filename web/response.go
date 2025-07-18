package web

import (
	"github.com/gin-gonic/gin"
)

type basicResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func Response(c *gin.Context, code int32, err string, data ...any) {
	var d any = nil
	if len(data) == 1 {
		d = data[0]
	}

	c.JSON(200, &basicResponse{
		Code:    code,
		Message: err,
		Data:    d,
	})
}
