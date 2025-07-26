package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TraceID 键名
const TraceIDKey = "traceId"

// goroutine local storage for TraceID
var (
	traceIDMap   = make(map[uint64]string)
	traceIDMutex = sync.RWMutex{}
)

// TraceIDWriter 自定义Writer，拦截并添加TraceID到日志
type TraceIDWriter struct {
	originalWriter io.Writer
	timeRegex      *regexp.Regexp
}

func NewTraceIDWriter(originalWriter io.Writer) *TraceIDWriter {
	return &TraceIDWriter{
		originalWriter: originalWriter,
		// 匹配Go标准log的时间戳格式：2025/07/12 11:24:30
		timeRegex: regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s`),
	}
}

func (w *TraceIDWriter) Write(p []byte) (n int, err error) {
	logLine := string(p)

	// 获取当前的TraceID
	traceID := GetTraceID()

	// 如果有TraceID，插入到时间戳后面
	if traceID != "" && w.timeRegex.MatchString(logLine) {
		logLine = w.timeRegex.ReplaceAllString(logLine, fmt.Sprintf("$1 【%s】", traceID))
	}

	// 写入到原始writer
	return w.originalWriter.Write([]byte(logLine))
}

// 生成TraceID
func GenerateTraceID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// 获取当前goroutine ID
func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]

	// 解析goroutine ID from stack trace
	// stack trace格式: "goroutine 123 [running]:"
	var gid uint64
	fmt.Sscanf(string(b), "goroutine %d ", &gid)
	return gid
}

// 设置当前goroutine的TraceID
func SetTraceID(traceID string) {
	gid := getGoroutineID()
	traceIDMutex.Lock()
	traceIDMap[gid] = traceID
	traceIDMutex.Unlock()
}

// 获取当前goroutine的TraceID
func GetTraceID() string {
	gid := getGoroutineID()
	traceIDMutex.RLock()
	traceID, exists := traceIDMap[gid]
	traceIDMutex.RUnlock()

	if !exists {
		return ""
	}
	return traceID
}

// 清理当前goroutine的TraceID
func ClearTraceID() {
	gid := getGoroutineID()
	traceIDMutex.Lock()
	delete(traceIDMap, gid)
	traceIDMutex.Unlock()
}

// TraceID Hook for logrus
type TraceIDHook struct{}

// Levels 返回适用的日志级别
func (hook *TraceIDHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire 在每次日志记录时触发
func (hook *TraceIDHook) Fire(entry *logrus.Entry) error {
	traceID := GetTraceID()
	if traceID != "" {
		entry.Data[TraceIDKey] = traceID
	}
	return nil
}

// 初始化TraceID系统
func InitTraceIDSystem() {
	// 1. 设置标准Go log包使用自定义Writer
	traceIDWriter := NewTraceIDWriter(os.Stdout)
	log.SetOutput(traceIDWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 2. 配置logrus使用自定义格式
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
		ForceColors:     false,
		DisableColors:   true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return "", fmt.Sprintf("%s:%d", f.File, f.Line)
		},
	})

	// 3. 启用logrus调用者信息
	logrus.SetReportCaller(true)

	// 4. 添加TraceID Hook到logrus
	logrus.AddHook(&TraceIDHook{})

	// 5. 设置logrus也输出到我们的自定义Writer
	logrus.SetOutput(traceIDWriter)

	log.Printf("TraceID系统初始化完成 - 支持标准log包和logrus双重输出")
}

// Gin中间件：TraceID处理
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取TraceID，如果没有则生成新的
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = GenerateTraceID()
		}

		// 设置当前goroutine的TraceID
		SetTraceID(traceID)

		// 将TraceID添加到Gin上下文中
		c.Set(TraceIDKey, traceID)

		// 将TraceID添加到响应头
		c.Header("X-Trace-ID", traceID)

		// 继续处理请求
		c.Next()

		// 请求处理完成后清理TraceID
		ClearTraceID()
	}
}

// 从Gin上下文获取TraceID
func GetTraceIDFromGin(c *gin.Context) string {
	if traceID, exists := c.Get(TraceIDKey); exists {
		return traceID.(string)
	}
	return ""
}

// 从标准context获取TraceID
func GetTraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// 将TraceID添加到标准context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// 便捷的日志函数，自动带TraceID
func LogInfo(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func LogError(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

func LogWarn(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

func LogDebug(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}
