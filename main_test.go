package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Qtag testing
// structured from http://www.golang-book.com/12
// XXX see if a full testsuite can be written (single test structure)
type testpair struct {
	s   []byte
	id  int64
	qts Qtags
}

var tests = []testpair{
	{
		[]byte("/!foo/bar!baz!lol/////xd!rofl/foo!/foo/test//"),
		-1,
		Qtags{
			{true, "bar"},
			{false, "baz"},
			{false, "foo"},
			{false, "lol"},
			{false, "rofl"},
			{true, "test"},
			{true, "xd"},
		},
	},
	{
		[]byte("/1"),
		1,
		Qtags{},
	},
	{
		[]byte("/!43!11"),
		-1,
		Qtags{
			{false, "11"},
			{false, "43"},
		},
	},
	{
		[]byte("/42/foo/baz/bar/whatsup/lol!foo"),
		42,
		Qtags{
			{true, "bar"},
			{true, "baz"},
			{true, "foo"},
			{true, "lol"},
			{true, "whatsup"},
		},
	},
}

// implicitely checking no duplicate and are lexically sorted
func TestProcessQtags(t *testing.T) {
	for _, pair := range tests {
		qts, id := ProcessQtags(pair.s)
		if id != pair.id {
			t.Error("Expecting id", pair.id, "but found", id)
		}
		if len(qts) != len(pair.qts) {
			t.Error("wrong number of results: expecting", len(pair.qts), "vs", len(qts))
		}
		for n := range qts {
			if qts[n].sign != pair.qts[n].sign || qts[n].name != pair.qts[n].name {
				t.Error("Expecting '", pair.qts[n], "' but found '", qts[n], "'")
			}
		}
	}
}

// Utils
func TestXint(t *testing.T) {
	if xInt("/3232a") != -1 || xInt("9001") != 9001 {
		t.Error("WONG!")
	}
}

// Storage testing. Also check some basic GetItems forms
var items = []Item{
	{
		1,
		"some content",
		"/foo/bar/baz",
	},
	{
		2,
		"more content",
		"/foo/bar/test",
	},
	{
		3,
		"blabla",
		"/baz/foo",
	},
	{
		4,
		"<encrypted content>",
		"/owner:mario",
	},
	{
		5,
		`{ "Id" : "1", "Content" : "Some JSON refering to an other Item" }`,
		"/owner:mario/type:json/",
	},
}

type testpair2 struct {
	s   []byte
	ids []int64
}

var getqueries = []testpair2{
	{
		[]byte("/2"),
		[]int64{2},
	},
	{
		[]byte("/!2!3"),
		[]int64{1, 4, 5},
	},
	{
		[]byte("/foo"),
		[]int64{1, 2, 3},
	},
	{
		[]byte("/!foo"),
		[]int64{4, 5},
	},
	{
		[]byte("/foo/bar!baz"),
		[]int64{2},
	},
	{
		[]byte("/foo/bar!baz/rofl"),
		[]int64{},
	},
}

// we test here:
//	- Database creation
//	- AddItem
//	- GetItems
func TestStorage1(t *testing.T) {
	f := "/tmp/test.sql"
	// add & get by id
	db := NewTags(f)
	defer os.Remove(f)

	if db == nil {
		t.Error("Database creation failure")
	}

	// AddItem
	for _, item := range items {
		qts, _ := ProcessQtags([]byte(item.Tags))
		id := db.AddItem(item.Content, qts)
		if id == -1 {
			t.Error("Item not added ", item)
		}
	}

	// GetItems
	for _, pair := range getqueries {
		its := db.GetItems(ProcessQtags(pair.s))
		if len(its) != len(pair.ids) {
			t.Error("Invalid number of fetched items for ", string(pair.s))
			t.Error(ProcessQtags(pair.s))
			t.Error(its)
		} else {
			// GetItems sort by id
			for i := range its {
				if pair.ids[i] != its[i].Id {
					t.Error("Expecting id", pair.ids[i], "vs", its[i])
				}
			}
		}
	}
}

var retags = []string{
	"/1!foo!baz/test",
	"/2/baz",
}

var retagqueries = []testpair2{
	{
		[]byte("/test"),
		[]int64{1, 2},
	},
	{
		[]byte("/baz"),
		[]int64{2, 3},
	},
}

// we test here:
//	- Database creation
//	- AddItem
//	- Retagging
//	- DeleteItems
func TestStorage2(t *testing.T) {
	f := "/tmp/test.sql"
	// add & get by id
	db := NewTags(f)
	defer os.Remove(f)

	if db == nil {
		t.Error("Database creation failure")
	}

	// AddItem
	for _, item := range items {
		qts, _ := ProcessQtags([]byte(item.Tags))
		id := db.AddItem(item.Content, qts)
		if id == -1 {
			t.Error("Item not added ", item)
		}
	}

	// Retagging/GetItems
	for _, r := range retags {
		qts, id := ProcessQtags([]byte(r))
		if id == -1 {
			t.Error("Invalid retag Qtag")
		} else {
			db.TagItem(id, qts)
		}
	}
	for _, pair := range retagqueries {
		its := db.GetItems(ProcessQtags(pair.s))
		if len(its) != len(pair.ids) {
			t.Error("Invalid number of fetched items for ", string(pair.s))
			t.Error(ProcessQtags(pair.s))
			t.Error(its)
		} else {
			// GetItems sort by id
			for i := range its {
				if pair.ids[i] != its[i].Id {
					t.Error("Expecting id", pair.ids[i], "vs", its[i])
				}
			}
		}
	}

	// DeleteItems
	for _, pair := range retagqueries {
		db.DeleteItems(ProcessQtags(pair.s))
		for _, id := range pair.ids {
			its := db.GetItems(ProcessQtags([]byte("/" + xItoa(id))))
			if len(its) != 0 {
				t.Error("Should have been deleted:", id)
			}
		}
	}
}

type JSONID struct {
	Id int64
}

// HTTP testing, re-use the data of TestStorage1/2
func TestHTTP(t *testing.T) {
	// Create Database
	f := "/tmp/test.sql"
	// db is in tags.go
	db = NewTags(f)
	defer os.Remove(f)

	if db == nil {
		t.Error("Database creation failure")
	}

	// Start HTTP server
	server := httptest.NewServer(http.HandlerFunc(tagHandler))
	defer server.Close()

	// POST to add a few items
	for _, item := range items {
		resp, err := http.Post(server.URL+"/"+item.Tags, "text/plain",
			bytes.NewBufferString(item.Content))
		if err != nil {
			t.Error("HTTP POST failed", err, "on", item)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Error on HTTP POST return value", err)
		}
		var Id JSONID
		err = json.Unmarshal(body, &Id)
		if err != nil {
			t.Error("JSON unmarshalling error:", err)
		}
		if Id.Id != item.Id {
			t.Error("Wrong ID: expecting ", item.Id, "vs", Id.Id)
			t.Error(string(body))
		}
	}

	// GET by ID to retrieve them, check JSON
	for _, item := range items {
		resp, err := http.Get(server.URL + "/" + xItoa(item.Id))
		if err != nil {
			t.Error("HTTP GET failed", err, "on", server.URL+"/"+xItoa(item.Id))
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Error on HTTP GET return value", err)
		}
		var it []Item
		err = json.Unmarshal(body, &it)
		if err != nil {
			t.Error("JSON unmarshalling error:", err)
		}
		if len(it) != 1 || it[0].Id != item.Id || it[0].Content != item.Content {
			t.Error("Wrong Item: expecting ", item, "vs", it[0])
			t.Error(string(body))
		}
	}

	// GET by Qtag
	for _, pair := range getqueries {
		resp, err := http.Get(server.URL + "/" + string(pair.s))
		if err != nil {
			t.Error("HTTP GET failed", err, "on", server.URL+"/"+string(pair.s))
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Error on HTTP GET return value", err)
		}
		var it []Item
		err = json.Unmarshal(body, &it)

		if len(it) != len(pair.ids) {
			t.Error("Wrong number of items: expecting", len(pair.ids), "vs", len(it))
		} else {
			for n := range it {
				if it[n].Id != pair.ids[n] {
					t.Error("Wrong Id: expecting", pair.ids[n], "vs", it[n].Id)
				}
			}
		}
	}

	// POST to retag items
	for _, r := range retags {
		resp, err := http.Post(server.URL+r, "text/plain",
			bytes.NewBufferString(""))

		if err != nil {
			t.Error("HTTP POST (retag) failed", err, "on", server.URL+r)
		}
		resp.Body.Close()
	}
	// GET to test previous retag
	for _, pair := range retagqueries {
		resp, err := http.Get(server.URL + "/" + string(pair.s))
		if err != nil {
			t.Error("HTTP GET (retag) failed", err, "on", server.URL+"/"+string(pair.s))
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error("Error on HTTP GET return value", err)
		}
		var it []Item
		err = json.Unmarshal(body, &it)

		if len(it) != len(pair.ids) {
			t.Error("Wrong number of items: expecting", len(pair.ids), "vs", len(it))
		} else {
			for n := range it {
				if it[n].Id != pair.ids[n] {
					t.Error("Wrong Id: expecting", pair.ids[n], "vs", it[n].Id)
				}
			}
		}
	}

	// DELETE items
	for _, pair := range retagqueries {
		req, err := http.NewRequest("DELETE", server.URL+"/"+string(pair.s), nil)
		if err != nil {
			t.Error("Error while creating DELETE request:", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error("Error while requesting DELETE:", err)
		}
		defer resp.Body.Close()

		for _, id := range pair.ids {
			resp2, err := http.Get(server.URL + "/" + xItoa(id))
			if err != nil {
				t.Error("Error with GET on", server.URL+"/"+xItoa(id))
			}
			defer resp2.Body.Close()
			body, err := ioutil.ReadAll(resp2.Body)
			if err != nil {
				t.Error("Error on HTTP GET return value", err)
			}
			var it []Item
			err = json.Unmarshal(body, &it)
			if len(it) != 0 {
				t.Error("Should have been deleted:", id)
			}
		}
	}
}
