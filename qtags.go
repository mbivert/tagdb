package main

import (
	"sort"
	"strings"
)

type Qtag struct {
	sign bool // true:tag is there, false:tag is not here
	name string
}

type Qtags []Qtag

func (i Qtag) String() string {
	if i.sign {
		return "/" + string(i.name)
	} else {
		return "!" + string(i.name)
	}
}

func (qt Qtags) String() (s string) {
	for _, i := range qt {
		s += i.String()
	}
	return
}

// for sort.Sort()
func (qt Qtags) Len() int           { return len(qt) }
func (qt Qtags) Swap(i, j int)      { qt[i], qt[j] = qt[j], qt[i] }
func (qt Qtags) Less(i, j int) bool { return string(qt[i].name) < string(qt[j].name) }

// assert(isop(s[0]))
func ProcessQtags(s []byte) (qts Qtags, id int64) {
	seen := make(map[string]bool)
	qts = make(Qtags, 0)
	id = -1

	j := 0

	for i := 1; i < len(s); {
		i += skipOp(s[i:])
		// nothing left but operators
		if i == len(s) {
			break
		}

		// s[j] is last operator
		j = i - 1
		// s[j+1:i] is tag associated to operator s[j]
		i += skipNotOp(s[i:])

		name := string(s[j+1 : i])

		if !seen[name] {
			seen[name] = true
			qts = append(qts, Qtag{s[j] == '/', name})
		}
		j = i
	}

	id2 := xInt(qts[0].name)
	// if first Qtag is a '/id', save it and
	// remove it from qts
	if qts[0].sign && id2 != -1 {
		qts = qts[1:]
		id = id2
	}

	sort.Sort(qts)

	return
}

func hasQtags(i Item, qts Qtags) bool {
	for _, qt := range qts {
		// checking for !id
		id := xInt(qt.name)
		// no need to check sign as remaining ids from ProcessQtags all
		// have a false sign
		if id != -1 && i.Id == id /* && !qt.sign */ {
			return false
		}
		// (sign && found) || (sign && !found)
		// <=> sign ^ found
		// <=> sign != found
		if qt.sign != strings.Contains(i.Tags, qt.name) {
			return false
		}
	}

	return true
}
