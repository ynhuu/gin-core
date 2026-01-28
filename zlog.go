package core

import (
	"errors"
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

func newZap(al zap.AtomicLevel) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	path, _ := os.Executable()
	name := fmt.Sprintf("logs/%s.log", filepath.Base(path))
	writer := lumberjack.Logger{
		Filename:   name, // 日志文件路径
		MaxSize:    20,   // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: 5,    // 日志文件最多保存多少个备份
		MaxAge:     5,    // 文件最多保存多少天
		Compress:   true, // 是否压缩
	}

	ws := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}
	if gin.Mode() == gin.ReleaseMode {
		ws = append(ws, zapcore.AddSync(&writer))
	}

	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.NewMultiWriteSyncer(ws...), al)

	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)

}

func ginLogger() gin.HandlerFunc {
	logger := zap.L().WithOptions(zap.WithCaller(false))
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" || c.Writer.Status() == 404 {
			c.Next()
			return
		}
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		cost := time.Since(start)
		fields := []zapcore.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", RealIP(c)),
			zap.Duration("cost", cost),
		}
		if query != "" {
			fields = append(fields, zap.String("query", query))
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				logger.Error(e, fields...)
			}
		} else {
			logger.Info("route", fields...)
		}
	}
}

func ginRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					var se *os.SyscallError
					if errors.As(ne, &se) {
						seStr := strings.ToLower(se.Error())
						if strings.Contains(seStr, "broken pipe") ||
							strings.Contains(seStr, "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					zap.L().Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					_ = c.Error(err.(error))
					c.Abort()
					return
				}

				if stack {
					zap.L().Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					zap.L().Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
