package commons

import (
	"bokee/internal/mods/interfaces"
	"fmt"

	"reflect"
	"strings"
)

// camelToSnake 将小驼峰转为蛇形：isTop → is_top，ASCII 字节级转换，零内存重分配
func camelToSnake(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s) + 2) // 预分配，最多加 2 个 _
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteByte(c + 32) // 大写 → 小写
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// StructToUpdateMap 将结构体的非空指针字段转为更新 map。
func StructToUpdateMap(data interface{}) map[string]interface{} {
	updates := make(map[string]interface{})
	v := reflect.ValueOf(data)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			tag := t.Field(i).Tag.Get("json")
			if tag == "" || tag == "-" {
				continue
			}
			tagName := strings.Split(tag, ",")[0]
			updates[camelToSnake(tagName)] = field.Elem().Interface()
		}
	}
	return updates
}

// CheckOwnership 校验当前用户是否拥有该资源，否则返回"无权操作"错误
func CheckOwnership(record interfaces.Ownable, userId uint) error {
	if record.GetUserID() != userId {
		return fmt.Errorf("无权操作他人资源")
	}
	return nil
}
