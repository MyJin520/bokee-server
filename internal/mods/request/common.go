package request

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

type PageReq struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

func (p *PageReq) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

func (p *PageReq) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// ParseQueryUint 仅负责将值转换为Uint类型
func ParseQueryUint(c *gin.Context, key string) (uint, bool) {
	str := c.Query(key)
	if str == "" {
		return 0, false
	}

	id, err := strconv.ParseUint(str, 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}

	return uint(id), true
}
