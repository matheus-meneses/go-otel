package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/matheus-meneses/go-otel/internal/pkg/storage"
	"github.com/matheus-meneses/go-otel/internal/pkg/trace"
	"github.com/matheus-meneses/go-otel/internal/users"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	ctx := context.Background()

	// Bootstrap tracer.
	prv, err := trace.NewProvider(trace.ProviderConfig{
		ProviderEndpoint: "127.0.0.1:4317",
		ServiceName:      "client",
		ServiceVersion:   "1.0.0",
		Environment:      "dev",
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer prv.Close(ctx)

	// Bootstrap database.
	dtb, err := sql.Open("mysql", "user:pass@tcp(:3306)/client")
	if err != nil {
		log.Fatalln(err)
	}
	defer dtb.Close()

	// Bootstrap API.
	usr := users.New(storage.NewUserStorage(dtb))

	// Bootstrap HTTP router.
	rtr := http.DefaultServeMux
	rtr.HandleFunc("/api/v1/users", trace.HTTPHandlerFunc(usr.Create, "users_create"))

	// Start HTTP server.
	if err := http.ListenAndServe(":8080", rtr); err != nil {
		log.Fatalln(err)
	}
}
