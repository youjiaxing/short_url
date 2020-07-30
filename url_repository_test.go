package main

import "testing"

func TestUrlRepository(t *testing.T) {
	//repos := NewUrlRepository(5, "")
	repos := NewUrlRepository(5, ":6379")

	v1,e1 := repos.Get("abc")
	t.Logf("%#q %#q\n", v1, e1)

	short, err := repos.Put("www.baidu.com")
	if err != nil {
		t.Error(err)
	}
	t.Log("short url:", short)

	long, err := repos.Get(short)
	if err != nil {
		t.Error(err)
	}
	t.Log("long url:", long)

	_, err = repos.Put("wm")
	if err != nil {
		t.Log(err)
	}

	err = repos.Delete(short)
	if err != nil {
		t.Error(err)
	}

	_, err = repos.Get(short)
	if err != nil {
		t.Log(err)
	}

	short, err = repos.Put("www.baidu2.com")
	if err != nil {
		t.Error(err)
	}
	t.Log("short url:", short)
}