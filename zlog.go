package core

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

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
		Filename:   name,
		MaxSize:    20,
		MaxBackups: 5,
		MaxAge:     5,
		Compress:   true,
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
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		if c.Writer.Status() == http.StatusNotFound {
			return
		}

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
				recoveredErr, ok := err.(error)
				if !ok {
					recoveredErr = fmt.Errorf("%v", err)
				}

				if errors.Is(recoveredErr, http.ErrAbortHandler) {
					c.Abort()
					return
				}

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
						zap.Error(recoveredErr),
						zap.String("request", string(httpRequest)),
					)
					_ = c.Error(recoveredErr)
					c.Abort()
					return
				}

				if stack {
					zap.L().Error("[Recovery from panic]",
						zap.Error(recoveredErr),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					zap.L().Error("[Recovery from panic]",
						zap.Error(recoveredErr),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
