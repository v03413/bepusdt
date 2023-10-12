package web

import "github.com/gin-gonic/gin"

type Response struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	RequestID  string      `json:"request_id"`
}

func RespFailJson(err error) gin.H {
	return gin.H{
		"status_code": 400,
		"message":     err.Error(),
	}
}

func RespSuccJson(data interface{}) gin.H {
	return gin.H{
		"status_code": 200,
		"message":     "success",
		"data":        data,
		"request_id":  "",
	}
}
