package main

import (
	"github.com/dragonsinth/produce/internal"
	"log"
	"net/http"
)

func main() {
	store := internal.NewStore()
	svc := internal.NewService(store)
	svc.Register(http.DefaultServeMux)
	log.Fatal(http.ListenAndServe("127.0.0.1:9000", nil))
}
