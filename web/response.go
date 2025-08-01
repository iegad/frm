package web

import (
	"encoding/base64"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/gox/frm/utils"
)

type basicResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
}

func Response(c *gin.Context, code int32, err string, data ...any) {
	var d []byte

	res := ""
	if len(data) > 0 {
		d, _ = json.Marshal(data[0])
		if len(data) > 1 {
			if key, ok := data[1].([]byte); ok {
				d, _ = utils.AesGcmEncrypt(d, key)
				res = base64.StdEncoding.EncodeToString(d)
			}
		}
	}

	c.JSON(200, &basicResponse{
		Code:    code,
		Message: err,
		Data:    res,
	})
}
