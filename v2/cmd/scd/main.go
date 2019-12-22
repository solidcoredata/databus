package main

import (
    "context"
    "time"
    "log"

    "github.com/kardianos/task"
)

func main() {
    ctx := context.Background()
    err := task.Start(ctx, time.Second * 3, run)
    if err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    log.Println("hello world")
    return nil
}