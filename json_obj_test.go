package pilot

import "testing"

type SampleObject struct {
	Name string
	Age  int
}

func TestJsonObject(t *testing.T) {
	json := []byte(`{"name":"John","age":30,"friends":[{"name":"Bob","age":20},{"name":"Alice","age":21}]}`)
	obj := NewJsonObject()
	err := obj.Parse(&json)
	if err != nil {
		t.Error(err)
	}
	name, err := obj.GetString("name")
	if *name != "John" {
		t.Error("Name is not John")
	}
	friends, err2 := obj.GetArray("friends")
	if err2 != nil {
		t.Error(err)
	}
	friend0, err3 := friends.GetObject(0)
	if err3 != nil {
		t.Error(err)
	}
	name, err = friend0.GetString("name")
	if *name != "Bob" {
		t.Error("Name is not Bob")
	}
	age, err := friend0.GetInt32("age")
	if *age != 20 {
		t.Error("Age is not 20")
	}
}
