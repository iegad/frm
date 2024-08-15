package web

import (
	"github.com/gin-gonic/gin"
)

type basicResponse struct {
	Code  int32       `json:"code"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func Response(c *gin.Context, code int32, err string, data interface{}) {
	c.JSON(200, &basicResponse{
		Code:  code,
		Error: err,
		Data:  data,
	})
}
