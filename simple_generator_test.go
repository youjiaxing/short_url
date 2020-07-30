package main

import "testing"

func TestGenerate(t *testing.T) {
	format := "%d %10s"
	for i := 0; i < 10; i++ {
		short := SimpleGen.gen(i)

		if len(short) != i {
			t.Errorf(format, i, short)
		} else {
			t.Logf(format, i, short)
		}
	}
}
