// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package columnar_test

import (
	"encoding/json"
	"sync"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/columnar"
	"go.xyrillian.de/gg/jsonmatch"
)

func testSuccessfulJSONRoundtrip[V any](t *testing.T, list []V, encoded jsonmatch.Diffable) {
	buf, err := json.Marshal(columnar.List[V](list))
	if err != nil {
		t.Fatal(err)
	}
	for _, diff := range encoded.DiffAgainst(buf) {
		t.Error(diff.String())
	}

	var unmarshaled columnar.List[V]
	err = json.Unmarshal(buf, &unmarshaled)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []V(unmarshaled), list)
}

func TestJSONRoundtripBasic(t *testing.T) {
	// try the example from the package docstring
	type Person struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Married   bool   `json:"married"`
	}

	testSuccessfulJSONRoundtrip(t,
		[]Person{
			{ID: 1, FirstName: "Alice", LastName: "Allison", Married: false},
			{ID: 2, FirstName: "Bob", LastName: "Burger", Married: true},
			{ID: 3, FirstName: "Carol", LastName: "Callagher", Married: true},
		},
		jsonmatch.Object{
			"id":         jsonmatch.Array{1, 2, 3},
			"first_name": jsonmatch.Array{"Alice", "Bob", "Carol"},
			"last_name":  jsonmatch.Array{"Allison", "Burger", "Callagher"},
			"married":    jsonmatch.Array{false, true, true},
		},
	)
}

func TestJSONRoundtripWithSpecialFields(t *testing.T) {
	type Point struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	type Event struct {
		*Point    `json:"location"` // embedded field, also with pointer
		Name      string            // without tag -> uses field name as key
		nameMutex sync.RWMutex      //nolint:unused // not exported -> ignored
	}

	testSuccessfulJSONRoundtrip(t,
		[]Event{
			{Point: &Point{X: 0, Y: 0}, Name: "origin"},
			{Point: &Point{X: 2, Y: 4}, Name: "somewhere"},
		},
		jsonmatch.Object{
			"location": jsonmatch.Array{jsonmatch.Object{"x": 0, "y": 0}, jsonmatch.Object{"x": 2, "y": 4}},
			"Name":     jsonmatch.Array{"origin", "somewhere"},
		},
	)
}

func TestJSONRoundtripResolvesPointers(t *testing.T) {
	type Record struct {
		Foo int
		Bar int
	}
	testSuccessfulJSONRoundtrip(t,
		[]***Record{
			new(new(new(Record{1, 2}))),
			new(new(new(Record{2, 4}))),
			new(new(new(Record{3, 6}))),
		},
		jsonmatch.Object{
			"Foo": jsonmatch.Array{1, 2, 3},
			"Bar": jsonmatch.Array{2, 4, 6},
		},
	)
}

func TestJSONRoundtripErrors(t *testing.T) {
	type nothingPublic struct {
		foo int
		bar int
	}

	_, err := json.Marshal(columnar.List[nothingPublic]{{1, 2}})
	assert.Equal(t, err.Error(), `json: error calling MarshalJSON for type columnar.List[go.xyrillian.de/gg/columnar_test.nothingPublic·5]: columnar_test.nothingPublic has no exported fields`)
	err = json.Unmarshal([]byte(`{}`), new(columnar.List[nothingPublic]{}))
	assert.Equal(t, err.Error(), `columnar_test.nothingPublic has no exported fields`)
}

func TestJSONUnmarshalFromInconsistentLengths(t *testing.T) {
	type Record struct {
		Foo int
		Bar int
	}

	err := json.Unmarshal([]byte(`{"Foo":[1,2],"Bar":[3,4,5]}`), new(columnar.List[Record]{}))
	assert.Equal(t, err.Error(), `cannot unmarshal from columns with inconsistent lengths [2 3]`)
}

func TestIgnoredFields(t *testing.T) {
	type Record struct {
		Foo   int `json:"foo"`
		Bonus int `json:"-"`
	}

	// There used to be a bug where columnar got confused that the `Bonus` field
	// ends up as a zero-length array after unmarshaling.
	testSuccessfulJSONRoundtrip(t,
		[]Record{
			{Foo: 1, Bonus: 0},
			{Foo: 2, Bonus: 0},
			{Foo: 3, Bonus: 0},
		},
		jsonmatch.Object{
			"foo": jsonmatch.Array{1, 2, 3},
		},
	)
}
