package common

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func GetUserRole(c *gin.Context) string {
	role, _ := c.Get("role")
	if role == nil {
		return ""
	}
	return role.(string)
}

func GetUserID(c *gin.Context) *uint {
	keys := []string{"user_id", "userID", "id", "uid", "userId"} // hardcoded
	
	for _, key := range keys {
		if value, exists := c.Get(key); exists {
			switch v := value.(type) {
			case uint:
				return &v
			case int, int64, uint64:
				id := uint(v.(int))
				return &id
			case float64:
				id := uint(v)
				return &id
			}
		}
	}
	return nil
}

func ToJSONString(v interface{}) *string {
	if v == nil {
		return nil
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	str := string(bytes)
	return &str
}

func GetStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}