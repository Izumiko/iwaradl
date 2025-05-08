package util

import (
	"fmt"
	"runtime"
	"time"
)

var Debug bool

// DebugLog 输出调试日志，包含时间戳和代码位置
func DebugLog(format string, v ...any) {
	if !Debug {
		return
	}

	// 获取时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 获取代码位置
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	// 获取文件名（不含路径）
	shortFile := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			shortFile = file[i+1:]
			break
		}
	}

	// 格式化消息
	message := fmt.Sprintf(format, v...)

	// 输出日志
	fmt.Printf("[%s] [%s:%d] %s\n", timestamp, shortFile, line, message)
}
