# gin-core

Small helpers around `gin` for server startup, request logging, panic recovery, real IP parsing, and YAML config loading.

## Features

- `MustNewServer(addr)` creates a `gin.Engine` with built-in logger and recovery middleware.
- `Start()` runs the HTTP server and handles graceful shutdown on `SIGINT`, `SIGTERM`, and `SIGTRAP`.
- Zap-based JSON logging.
- Request logs skip `OPTIONS` and `404` requests.
- `RealIP()` reads `Cf-Connecting-Ip`, `X-Forwarded-For`, then falls back to `c.ClientIP()`.
- `ReadYml[T]()` loads YAML config into a typed struct.

## Install

```bash
go get github.com/ynhuu/gin-core@latest
```

## Quick Start

```go
package main

import (
	"net/http"

	core "github.com/ynhuu/gin-core"
	"github.com/gin-gonic/gin"
)

func main() {
	server := core.MustNewServer(":8080")

	server.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	server.Start()
}
```

## Log Behavior

- Debug mode sets the zap level to `debug`.
- Release mode writes logs to stdout and `logs/<binary>.log`.
- Non-release mode writes logs to stdout only.
- `404` requests do not produce route logs.

## Read YAML

```go
package main

import (
	"log"

	core "github.com/ynhuu/gin-core"
)

type Config struct {
	Addr string `yaml:"addr"`
}

func main() {
	cfg, err := core.ReadYml[Config]("config.yml")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(cfg.Addr)
}
```

## API

```go
func MustNewServer(addr string) *Server
func (s *Server) SetZapLevel(l zapcore.Level)
func (s *Server) Start()
func RealIP(c *gin.Context) string
func ReadYml[T any](name string) (T, error)
```
