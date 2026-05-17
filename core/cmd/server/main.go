package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"distributed-object-storage/internal/config"
	"distributed-object-storage/internal/httpapi"
	"distributed-object-storage/internal/migrations"
	"distributed-object-storage/internal/replication"
	"distributed-object-storage/internal/repository"
	"distributed-object-storage/internal/service"
	"distributed-object-storage/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DBConnString)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err = migrations.New(db).Apply(); err != nil {
		log.Fatal(err)
	}

	repo := repository.New(db)

	store := storage.NewLocal(cfg.StorageNodes)
	if err = store.Init(); err != nil {
		log.Fatal(err)
	}

	objects := service.NewObjectService(repo, store)
	worker := replication.NewWorker(repo, store)
	go worker.Run()

	handler := httpapi.NewHandler(objects, cfg.AppSecret)
	mux := http.NewServeMux()
	handler.Register(mux)

	if err = http.ListenAndServe(cfg.HTTPAddr, mux); err != nil {
		log.Fatal(err)
	}
}
