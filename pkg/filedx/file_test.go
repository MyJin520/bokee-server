package filedx

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestFile(t *testing.T) {
	files := []string{
		"xxxxx",
	}
	for _, v := range files {
		file, err := os.ReadFile(v)
		if err != nil {
			fmt.Println(v, "读取失败")
			continue
		}
		fmt.Println("=================================================")
		mime, err := GetFileMIME(bytes.NewReader(file))
		if err != nil {
			fmt.Println(v, "获取文件类型失败", err.Error())
			continue
		}
		allowe := CheckIfAllowed(mime.MIME.Value)
		if !allowe {
			fmt.Println(v, "不允许的文件")
		} else {
			fmt.Println("允许的文件")
		}

		//fmt.Println("文件类型：", ext.Extension)
	}
}
