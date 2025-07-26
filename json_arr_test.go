package pilot

import "testing"

func TestJsonArr(t *testing.T) {
	json := []byte(`[1,2,3,4]`)
	obj := NewJsonArray()
	err := obj.Parse(json)
	if err != nil {
		t.Error(err)
	}
	val1, err2 := obj.GetInt64(0)
	if err2 != nil {
		t.Error(err2)
	}
	if *val1 != 1 {
		t.Error("val1 != 1")
	}
}
