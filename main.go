package main

import (
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	initDB()

	http.HandleFunc("/upload/", uploadHandler)

	http.HandleFunc("/download/", downloadHandler)

	http.HandleFunc("/presign/", presignHandler)

	http.ListenAndServe(":8080", nil)
}
