package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("simulate data bus")
	a := &app{}
	http.ListenAndServe(":8080", a)
}

type app struct{}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("I'm alive!"))
}

