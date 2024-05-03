package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 3000, "HTTP Port")
	flag.Parse()

	app := newApp(port)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/files", app.handlerFiles)
	mux.HandleFunc("GET /api/users", app.handlerUsers)

	app.serve(mux)
}

type app struct {
	data   []map[string]string
	logger *slog.Logger
	mu     sync.Mutex
	port   int
}

func (app *app) serve(mux *http.ServeMux) {
	app.logger.Info("listening", "port", app.port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", app.port), mux)
	if err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}
}

func newApp(port int) *app {
	return &app{
		data:   []map[string]string{},
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		port:   port,
	}
}

func (app *app) responseError(w http.ResponseWriter, err error) {
	app.logger.Error(err.Error())
	code := http.StatusInternalServerError
	http.Error(w, http.StatusText(code), code)
}

type envelope map[string]any

func (app *app) sendJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) {
	js, err := json.Marshal(data)
	if err != nil {
		app.responseError(w, err)
		return
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(js); err != nil {
		app.responseError(w, err)
	}
}

func (app *app) handlerFiles(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10_485_760)

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		if errors.Is(err, io.EOF) {
			app.sendJSON(w, http.StatusBadRequest, envelope{"message": "A file must be provided"}, nil)
			return
		}

		app.responseError(w, err)
		return
	}
	defer file.Close()

	if fileHeader.Header.Get("Content-Type") != "text/csv" {
		app.sendJSON(w, http.StatusBadRequest, envelope{"message": "The file type must be CSV"}, nil)
		return
	}

	reader := csv.NewReader(file)
	keys, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			app.sendJSON(w, http.StatusBadRequest, envelope{"message": "The file must not be empty"}, nil)
			return
		}

		app.responseError(w, err)
		return
	}

	rows, err := reader.ReadAll()
	if err != nil {
		app.responseError(w, err)
		return
	}
	if len(rows) == 0 {
		app.sendJSON(w, http.StatusBadRequest, envelope{
			"message": "Send a file with records",
		}, nil)
		return
	}

	app.mu.Lock()

	if len(app.data) > 0 {
		app.data = nil
	}

	for _, cell := range rows {
		item := make(map[string]string)

		for index, key := range keys {
			item[key] = cell[index]
		}

		app.data = append(app.data, item)
	}

	app.mu.Unlock()

	app.sendJSON(w, http.StatusOK, envelope{
		"message": "File uploaded successfully",
	}, nil)
}

func (app *app) handlerUsers(w http.ResponseWriter, r *http.Request) {
	app.mu.Lock()
	defer app.mu.Unlock()

	if len(app.data) == 0 {
		app.sendJSON(w, http.StatusNotFound, envelope{
			"message": "There is not data, upload a CSV file first",
		}, nil)
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		app.sendJSON(w, http.StatusOK, envelope{
			"data": app.data,
		}, nil)
		return
	}

	q = strings.ToLower(q)
	var coincidences []map[string]string

	for _, record := range app.data {
		for _, v := range record {
			if strings.Index(strings.ToLower(v), q) != -1 {
				coincidences = append(coincidences, record)
			}
		}
	}

	app.sendJSON(w, http.StatusOK, envelope{
		"data": coincidences,
	}, nil)
}
