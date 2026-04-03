package plog

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync/atomic"
)

type Level int64

const (
	LevelDebug Level = 1 << (iota + 2) // 4
	LevelInfo                          // 8
	LevelWarn                          // 16
	LevelError                         // 32
)

// 定义 ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// String 提供带颜色的等级前缀
func (l Level) String() string {
	switch l {
	case LevelDebug:
		// 青色 (Cyan)
		return colorCyan + "[DEBUG]" + colorReset
	case LevelInfo:
		// 绿色 (Green)
		return colorGreen + "[INFO]" + colorReset
	case LevelWarn:
		// 黄色 (Yellow)
		return colorYellow + "[WARN]" + colorReset
	case LevelError:
		// 红色 (Red)
		return colorRed + "[ERROR]" + colorReset
	default:
		return "[LOG]"
	}
}

var defaultLevel atomic.Int64

func init() {
	defaultLevel.Store(int64(LevelInfo))
}

func SetLevel(level Level) {
	defaultLevel.Store(int64(level))
}

func enable(level Level) bool {
	return int64(level) >= defaultLevel.Load()
}

func printLog(level Level, skip int, msgS ...any) {
	if !enable(level) {
		return
	}

	// 获取调用栈信息
	_, file, line, ok := runtime.Caller(skip)
	caller := "???:0"
	if ok {
		// 截取干净的文件名
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line) + "\t"
		//caller = fmt.Sprintf("%s:%d", file, line)
	}

	args := make([]any, 0, len(msgS)+2)
	args = append(args, level.String(), caller)
	args = append(args, msgS...)

	fmt.Println(args...)
}

func Log(level Level, msgS ...any) {
	printLog(level, 2, msgS...)
}

func Info(msgS ...any) {
	printLog(LevelInfo, 2, msgS...)
}

func Warn(msgS ...any) {
	printLog(LevelWarn, 2, msgS...)
}

func Error(msgS ...any) {
	printLog(LevelError, 2, msgS...)
}

func Debug(msgS ...any) {
	printLog(LevelDebug, 2, msgS...)
}
