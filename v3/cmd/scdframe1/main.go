package main

import (
	"context"
	"log"
	"time"

	"github.com/kardianos/task"
)

func main() {
	ctx := context.Background()
	err := task.Start(ctx, time.Second*3, run)
	if err != nil {
		log.Fatal(err)
	}
}
