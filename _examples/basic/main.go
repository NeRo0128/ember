package main

import (
	"bytes"
	"fmt"

	"github.com/NeRo0128/ember/logger"
)

func main() {
	log := logger.NewLogger(
		logger.WithLevel(logger.InfoLevel),
		logger.WithJSON(true),
	)

	var buf bytes.Buffer
	log.AddOutput(&buf)

	log.Info("server started",
		logger.Field{Key: "port", Value: 8080},
		logger.Field{Key: "env", Value: "production"},
	)

	fmt.Println(buf.String())
}
