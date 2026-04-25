package core

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	Server struct {
		*gin.Engine
		addr string
		al   zap.AtomicLevel
	}
)

func MustNewServer(addr string) *Server {
	al := zap.NewAtomicLevelAt(zap.InfoLevel)
	newZap(al)
	server := &Server{
		gin.New(),
		addr,
		al,
	}
	server.Use(ginLogger(), ginRecovery(true))

	if gin.Mode() == gin.DebugMode {
		server.SetZapLevel(zap.DebugLevel)
	}

	return server
}

func (s *Server) SetZapLevel(l zapcore.Level) {
	s.al.SetLevel(l)
}

func (s *Server) Start() {
	srv := &http.Server{
		Handler: s,
		Addr:    s.addr,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTRAP, syscall.SIGTERM)
	<-quit
	zap.L().Info("Shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server shutdown", zap.Error(err))
	}
	zap.L().Info("Server exiting")
}
