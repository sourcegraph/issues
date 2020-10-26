package rcache

import (
	"reflect"
	"strconv"
	"testing"
)

func TestCache_namespace(t *testing.T) {
	SetupForTest(t)

	type testcase struct {
		prefix  string
		entries map[string]string
	}

	cases := []testcase{
		{
			prefix: "a",
			entries: map[string]string{
				"k0": "v0",
				"k1": "v1",
				"k2": "v2",
			},
		}, {
			prefix: "b",
			entries: map[string]string{
				"k0": "v0",
				"k1": "v1",
				"k2": "v2",
			},
		}, {
			prefix: "c",
			entries: map[string]string{
				"k0": "v0",
				"k1": "v1",
				"k2": "v2",
			},
		},
	}

	caches := make([]*Cache, len(cases))
	for i, test := range cases {
		caches[i] = New(test.prefix)
		for k, v := range test.entries {
			caches[i].Set(k, []byte(v))
		}
	}
	for i, test := range cases {
		// test all the keys that should be present are found
		for k, v := range test.entries {
			b, ok := caches[i].Get(k)
			if !ok {
				t.Fatalf("error getting entry from redis (prefix=%s)", test.prefix)
			}
			if string(b) != v {
				t.Errorf("expected %s, got %s", v, string(b))
			}
		}

		// test not found case
		if _, ok := caches[i].Get("not-found"); ok {
			t.Errorf("expected not found")
		}
	}
}

func TestCache_simple(t *testing.T) {
	SetupForTest(t)

	c := New("some_prefix")
	_, ok := c.Get("a")
	if ok {
		t.Fatal("Initial Get should find nothing")
	}

	c.Set("a", []byte("b"))
	b, ok := c.Get("a")
	if !ok {
		t.Fatal("Expect to get a after setting")
	}
	if string(b) != "b" {
		t.Fatalf("got %v, want %v", string(b), "b")
	}

	c.Delete("a")
	_, ok = c.Get("a")
	if ok {
		t.Fatal("Get after delete should of found nothing")
	}
}

func TestCache_multi(t *testing.T) {
	SetupForTest(t)

	c := New("some_prefix")
	vals := c.GetMulti("k0", "k1", "k2")
	if got, exp := vals, [][]byte{nil, nil, nil}; !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v on initial fetch, got %v", exp, got)
	}

	c.Set("k0", []byte("b"))
	if got, exp := c.GetMulti("k0"), bytes("b"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}

	c.SetMulti([2]string{"k0", "a"})
	if got, exp := c.GetMulti("k0"), bytes("a"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}

	c.SetMulti([2]string{"k0", "a"}, [2]string{"k1", "b"})
	if got, exp := c.GetMulti("k0"), bytes("a"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}
	if got, exp := c.GetMulti("k1"), bytes("b"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}
	if got, exp := c.GetMulti("k0", "k1"), bytes("a", "b"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}
	if got, exp := c.GetMulti("k1", "k0"), bytes("b", "a"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}

	c.SetMulti([2]string{"k0", "x"}, [2]string{"k1", "y"}, [2]string{"k2", "z"})
	if got, exp := c.GetMulti("k0", "k1", "k2"), bytes("x", "y", "z"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}
	got, exist := c.Get("k0")
	if exp := "x"; !exist || string(got) != exp {
		t.Errorf("Expected %v, but got %v", exp, string(got))
	}

	c.Delete("k0")
	if got, exp := c.GetMulti("k0", "k1", "k2"), [][]byte{nil, []byte("y"), []byte("z")}; !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}
}

func TestCache_deleteKeysWithPrefix(t *testing.T) {
	SetupForTest(t)

	// decrease the deleteBatchSize
	oldV := deleteBatchSize
	deleteBatchSize = 2
	defer func() { deleteBatchSize = oldV }()

	c := New("some_prefix")
	var aKeys, bKeys []string
	var key string
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			key = "a:" + strconv.Itoa(i)
			aKeys = append(aKeys, key)
		} else {
			key = "b:" + strconv.Itoa(i)
			bKeys = append(bKeys, key)
		}

		c.SetMulti([2]string{key, strconv.Itoa(i)})
	}

	conn := pool.Get()
	defer conn.Close()

	err := deleteKeysWithPrefix(conn, c.rkeyPrefix()+"a")
	if err != nil {
		t.Error(err)
	}

	vals := c.GetMulti(aKeys...)
	if got, exp := vals, [][]byte{nil, nil, nil, nil, nil}; !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}

	vals = c.GetMulti(bKeys...)
	if got, exp := vals, bytes("1", "3", "5", "7", "9"); !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, but got %v", exp, got)
	}
}

func bytes(s ...string) [][]byte {
	if s == nil {
		return nil
	}
	t := make([][]byte, len(s))
	for i, v := range s {
		t[i] = []byte(v)
	}
	return t
}
