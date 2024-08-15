package test

import (
	"testing"

	"github.com/gox/frm/utils"
)

type Person struct {
	Name   string
	Age    int64
	Remark [1024]byte
}

func TestMemory(t *testing.T) {
	a := []byte{1, 2, 3, 4, 5, 6}
	b := []byte{1, 2, 3, 4, 5, 6}

	t.Log(utils.Memcmp(&a, &b))

	c := utils.CloneSlice(a)

	a[1] = 10
	t.Log(c)
	t.Log(a)
}

func BenchmarkMemcmp(b *testing.B) {
	buf1 := &Person{
		Name:   "iegad",
		Age:    888,
		Remark: [1024]byte{},
	}
	buf2 := &Person{
		Name:   "iegad",
		Age:    888,
		Remark: [1024]byte{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !utils.Memcmp(buf1, buf2) {
			b.Fatal("failed")
		}
	}
}
