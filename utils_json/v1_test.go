package utils_json

import "testing"

func TestJSONEncodeDecode(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	in := payload{Name: "test", N: 42}
	s := JSONEncode(in)
	if s == "" {
		t.Fatal("JSONEncode returned empty")
	}

	var out payload
	if err := JSONDecode(s, &out); err != nil {
		t.Fatalf("JSONDecode: %v", err)
	}
	if out.Name != "test" || out.N != 42 {
		t.Fatalf("got %+v", out)
	}
}

func TestJSONEncodeE(t *testing.T) {
	s, err := JSONEncodeE(map[string]int{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if s != `{"a":1}` {
		t.Fatalf("got %s", s)
	}

	_, err = JSONEncodeE(make(chan int))
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestMapFilterCopy(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2}
	out := MapFilter(input, 10)
	out["c"] = 3
	if _, ok := input["c"]; ok {
		t.Fatal("MapFilter should return a copy")
	}
}
