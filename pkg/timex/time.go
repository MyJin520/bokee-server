package timex

import (
	"fmt"
	"strconv"
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	var total time.Duration
	var numStr string

	for i, ch := range s {
		if (ch >= '0' && ch <= '9') || ch == '.' {
			numStr += string(ch)
			continue
		}

		// 遇到单位字符
		if ch == 'd' || ch == 'h' || ch == 'm' || ch == 's' {
			if len(numStr) == 0 {
				return 0, fmt.Errorf("无效的持续时间：单位前缺少数字 %c", ch)
			}
			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("无效的持续时间：数字 %q 格式错误 %w %c", numStr, err, ch)
			}

			var unit time.Duration
			switch ch {
			case 'd':
				unit = time.Duration(val * 24 * float64(time.Hour))
			case 'h':
				unit = time.Duration(val * float64(time.Hour))
			case 'm':
				unit = time.Duration(val * float64(time.Minute))
			case 's':
				unit = time.Duration(val * float64(time.Second))
			}
			total += unit
			numStr = "" // 重置，准备下一个数字
		} else {
			return 0, fmt.Errorf("无效的持续时间：单位字符错误 %c", ch)
		}

		if i == len(s)-1 && len(numStr) > 0 {
			return 0, fmt.Errorf("无效的持续时间：尾随数字缺少单位字符 %q", numStr)
		}
	}

	if total == 0 {
		return 0, fmt.Errorf("无效的持续时间：持续时间为0")
	}
	return total, nil
}
