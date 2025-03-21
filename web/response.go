package web

import (
	"github.com/gin-gonic/gin"
)

type basicResponse struct {
	Code  int32       `json:"Code"`
	Error string      `json:"Error,omitempty"`
	Data  interface{} `json:"Data,omitempty"`
}

func Response(c *gin.Context, code int32, err string, data ...any) {
	var d any = nil
	if len(data) == 1 {
		d = data[0]
	}

	c.JSON(200, &basicResponse{
		Code:  code,
		Error: err,
		Data:  d,
	})
}
