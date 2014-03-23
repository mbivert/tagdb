// XXX check memory usage (string function, etc.)
// HTTP interface to access TagDB. Output JSON.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"unsafe"
)

var dbfile = flag.String("dbfile", "/tmp/db.sql", "SQLite Database file")
var port = flag.String("port", "8080", "Listening HTTP port")

var db *Database

func get(w http.ResponseWriter, r *http.Request) {
	s := []byte(r.URL.Path)
	qts, id := ProcessQtags(s)
	items := db.GetItems(qts, id)
	js, _ := json.Marshal(items)
	w.Write(js)
}

func post(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	b := buf.Bytes()
	content := *(*string)(unsafe.Pointer(&b))
	s := []byte(r.URL.Path)
	qts, id := ProcessQtags(s)

	// updating an item
	if id != -1 {
		db.TagItem(id, qts)
		if content != "" {
			db.UpdateContent(id, content)
		}
	} else {
		id = db.AddItem(content, qts)
	}

	fmt.Fprintf(w, "{ \"Id\":%d }", id)
}

func delete(w http.ResponseWriter, r *http.Request) {
	s := []byte(r.URL.Path)
	qts, id := ProcessQtags(s)

	db.DeleteItems(qts, id)
}

func tagHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		get(w, r)
	case "POST":
		post(w, r)
	case "DELETE":
		delete(w, r)
	}
}

func main() {
	db = NewTags(*dbfile)
	http.HandleFunc("/", tagHandler)

	log.Print("Launching on http://localhost:" + *port)

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
