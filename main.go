package main

import (
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	initDB()

	err := initStorage()
	if err != nil {
		panic(err)
	}

	initReplicationQueue()

	go replicationWorker()

	http.HandleFunc("/upload/", uploadHandler)

	http.HandleFunc("/download/", downloadHandler)

	http.HandleFunc("/presign/", presignHandler)

	http.ListenAndServe(":8080", nil)
}
