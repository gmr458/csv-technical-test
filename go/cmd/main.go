package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"strings"
)

var globalData []map[string]string

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/files", handlerFiles)
	mux.HandleFunc("GET /api/users", handlerUsers)

	log.Println("listening on port 3000")
	err := http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func handlerFiles(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10_485_760)

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		sendJSON(w, http.StatusBadRequest, envelope{"message": "A file must be provided"}, nil)
		return
	}
	defer file.Close()

	if fileHeader.Header.Get("Content-Type") != "text/csv" {
		sendJSON(w, http.StatusBadRequest, envelope{"message": "The file type must be CSV"}, nil)
		return
	}

	reader := csv.NewReader(file)
	keys, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			sendJSON(w, http.StatusBadRequest, envelope{"message": "The file must not be empty"}, nil)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := reader.ReadAll()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(rows) == 0 {
		sendJSON(w, http.StatusBadRequest, envelope{
			"message": "Send a file with records",
		}, nil)
		return
	}

	if len(globalData) > 0 {
		globalData = nil
	}

	for _, cell := range rows {
		item := make(map[string]string)

		for index, key := range keys {
			item[key] = cell[index]
		}

		globalData = append(globalData, item)
	}

	sendJSON(w, http.StatusOK, envelope{
		"message": "File uploaded successfully",
	}, nil)
}

func handlerUsers(w http.ResponseWriter, r *http.Request) {
	if len(globalData) == 0 {
		sendJSON(w, http.StatusNotFound, envelope{
			"message": "There is not data, upload a CSV file first",
		}, nil)
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		sendJSON(w, http.StatusOK, envelope{
			"data": globalData,
		}, nil)
		return
	}

	q = strings.ToLower(q)
	var coincidences []map[string]string

	for _, record := range globalData {
		for _, v := range record {
			if strings.Index(strings.ToLower(v), q) != -1 {
				coincidences = append(coincidences, record)
			}
		}
	}

	sendJSON(w, http.StatusOK, envelope{
		"data": coincidences,
	}, nil)
}
