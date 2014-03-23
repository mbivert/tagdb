// using sqlite for simplicity (prototyping)
// one may change for something else (other sql, flat files, etc.)
// or add a layer to support multiple storages system.

package main

import (
	"log"
	"runtime"
	"github.com/kuroneko/gosqlite3"
)

// Re use sqlite3 database; append it our own functions.
type Database struct {
	*sqlite3.Database
}

// Prepare(s, v), Step() and error checking
func (db *Database)Execute2(s string, v... interface{}) (stmt *sqlite3.Statement, err error) {
	stmt, err = db.Prepare(s, v...)
	if err == nil {
		err = stmt.Step()
	}
	if err != nil && err != sqlite3.ROW {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("Error with SQLite: %s, at %s:%d\n", err, file, line)
	}
	return
}

// Create a table described by descr
func (db *Database) CreateTable(descr string) {
	_, err  := db.Execute2(descr)
	if err != nil {
		log.Fatal("Error when creating SQLite tables:", err)
		return
	}
}

// Create a new Database from a file
func NewTags(filename string) (db *Database) {
	tmp, err := sqlite3.Open(filename)
	if err != nil {
		log.Fatal("Cannot open sqlite connection:", err)
	}
	db = &Database{tmp}

	db.CreateTables()

	return
}

// Create tables for TagsDB; -1 as invalid id for all tables
func (db *Database) CreateTables() {
	db.Execute2("PRAGMA foreign_keys = ON")
	db.Execute2(`create table if not exists tags (
		id			integer		primary key autoincrement,
		name		text		unique)`)
	db.Execute2(`create table if not exists items (
		id			integer		primary key autoincrement,
		content		text)`)
	db.Execute2(`create table if not exists mapping (
		id			integer		primary key autoincrement,
		idt			integer		not null,
		idi			integer		not null,
		foreign key	(idt)		references tags(id) on delete cascade,
		foreign key	(idi)		references items(id) on delete cascade)`)

	// -1 is invalid id for all.
	db.Execute2("BEGIN IMMEDIATE TRANSACTION")
		db.Execute2("insert or ignore into tags(id) values (-1)")
		db.Execute2("insert or ignore into items(id) values (-1)")
		db.Execute2("insert or ignore into mapping(id, idt, idi) values (-1, -1, -1)")
	db.Execute2("COMMIT TRANSACTION")
}

// Item to communicate value from/to db/dbuser
type Item struct {
	Id		int64
	Content	string
	Tags	string
}

// Retrieve an Item and its tag
func (db *Database) GetItem(id int64) (i Item) {
	i.Id = -1

	stmt, err := db.Execute2(`select items.content, group_concat(tags.name, '/')
			from tags, items, mapping
			where
				items.id = (?)
				and items.id = mapping.idi
				and tags.id  = mapping.idt`, id)

	v := stmt.Row()

	if err == sqlite3.ROW && v != nil && v[0] != nil {
		i = Item{ id, v[0].(string), v[1].(string) }
	}
	return
}

// Retrieve a tag id; insert it if it doesn't exists
func (db *Database) AddTag(name string) int64 {
	db.Execute2("insert or ignore into tags(name) values(?)", name)
	stmt, _ := db.Execute2("select id from tags where name = (?)", name)
	if stmt.Row() != nil {
		return (stmt.Row())[0].(int64)
	}
	return -1
}

func (db *Database) UpdateContent(id int64, content string) {
	db.Execute2("update items set content='(?)' where id=(?)", content, id)
}

// Add an Item
func (db *Database) AddItem(content string, qts Qtags) (id int64) {
	db.Execute2("insert into items(content) values (?)", content)
	id = db.LastInsertRowID()
	db.TagItem(id, qts)

	return
}

// Change tagging of an item from a querytag
func (db *Database) TagItem(id int64, qtags Qtags) {
	for qt := range qtags {
		idt := db.AddTag(string(qtags[qt].name))

		if qtags[qt].sign {
			db.Execute2("insert into mapping(idt, idi) values (?, ?)", idt, id)
		} else {
			db.Execute2("delete from mapping where idt=(?) and idi=(?)", idt, id)
		}
	}
}

// Retrieve set of items from a querytag
// XXX add a memory cache here to avoid The Query.
// XXX for instance cache query and its results
// XXX array of *Item instead for caching.
func (db *Database) GetItems(qts Qtags, id int64) (items []Item) {
	items = make([]Item, 0)

	// Qtags is related to a single Item
	if id != -1 {
		i := db.GetItem(id)
		if i.Id != -1 && hasQtags(i, qts) {
			items = append(items, i)
		}
		return
	}

	// Qtags is related to many items.
	s := `select items.id, items.content, group_concat(tags.name, '/')
		from tags, items, mapping
		where
			mapping.idt = tags.id
			and mapping.idi = items.id
			and items.id != -1
		group by items.id`

	stmt, err := db.Prepare(s)
	if err != nil {
		return
	}

	stmt.All(func (s *sqlite3.Statement, v... interface{}) {
		i := Item{ v[0].(int64), v[1].(string), v[2].(string) }
		if hasQtags(i, qts) {
			items = append(items, i)
		}
	})

	return
}

func (db *Database) DeleteItems(qts Qtags, id int64) {
	items := db.GetItems(qts, id)

	for _, item := range items {
		db.Execute2("delete from items where id = (?)", item.Id)
	}
}
