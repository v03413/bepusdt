package web

import "github.com/gin-gonic/gin"

func respFailJson(message string) gin.H {

	return gin.H{"status_code": 400, "message": message}
}

func respSuccJson(data interface{}) gin.H {

	return gin.H{"status_code": 200, "message": "success", "data": data, "request_id": ""}
}
