package main

import (
	"fmt"
	"time"

	"github.com/NeRo0128/ember/logger"
)

func main() {
	log := logger.NewLogger(
		logger.WithLevel(logger.InfoLevel),
		logger.WithAsync(100),
	)

	log.Info("async message")
	time.Sleep(100 * time.Millisecond) // wait for worker

	if err := log.Close(); err != nil {
		fmt.Println("close error:", err)
	}

	fmt.Println("done")
}
