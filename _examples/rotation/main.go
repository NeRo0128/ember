package main

import (
	"fmt"
	"os"

	"github.com/NeRo0128/ember/logger"
)

func main() {
	rotator, err := logger.NewRotatingFile("/tmp/ember-example.log", 1, 3, 0)
	if err != nil {
		panic(err)
	}

	log := logger.NewLogger(
		logger.WithLevel(logger.DebugLevel),
	)
	log.AddOutput(rotator)

	log.Info("application started")
	log.Debug("debug details")

	if err := log.Close(); err != nil {
		fmt.Println("close error:", err)
	}

	content, _ := os.ReadFile("/tmp/ember-example.log")
	fmt.Println(string(content))
}
