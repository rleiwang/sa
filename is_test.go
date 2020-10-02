package sa

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"testing"
	"time"
)

var datafile string

func init() {
	flag.StringVar(&datafile, "file", "", "testing data file")
}

func count(bwt []byte, c byte, o, l int) int {
	j := 0
	for i := 0; i <= o; i++ {
		if bwt[i] == c {
			j++
		}
	}

	return j
}

func rewindBWT(bwt []byte, l int) []byte {
	hist, bkt := histgram(bytebuf(bwt), 256)
	setBktBeg(bkt, hist)

	t := make([]byte, len(bwt)-1)

	for i, j := 0, 0; bwt[i] != 0; j++ {
		t[j] = bwt[i]
		if t[j] == 1 {
			i = bkt[1]
			bkt[1]++
		} else {
			c := count(bwt, bwt[i], i, l)
			i = bkt[bwt[i]] + c - 1
		}
	}

	return t
}

func toString(b []byte, o, r byte) string {
	for i := range b {
		if b[i] == o {
			b[i] = r
		}
	}

	return string(b)
}

func toByte(s string, o, r byte) []byte {
	b := []byte(s)

	for i := range b {
		if b[i] == o {
			b[i] = r
		}
	}

	return b
}

func Test_bwt(t *testing.T) {
	type args struct {
		t []byte
		l int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"name1", args{[]byte("abcabca"), 3}, []byte{'a', 'b', 'b', 0, 'c', 'c', 'a', 'a'}},
		{"name2", args{[]byte("ippississim"), 5}, []byte{'i', 'p', 's', 's', 'm', 0, 'p', 'i', 's', 's', 'i', 'i'}},
		{"name3", args{[]byte("iippiissiissiimm"), 10}, []byte{'i', 'i', 'p', 's', 's', 'm', 'i', 'i', 'i', 'm', 0, 'p', 'i', 's', 's', 'i', 'i'}},
		{"name4", args{[]byte("sisisisim"), 5}, []byte{'s', 's', 's', 's', 'm', 0, 'i', 'i', 'i', 'i'}},
		{"name6", args{toByte("ananab$abana$nana", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name5", args{toByte("nana$abana$ananab", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name7", args{toByte("sisim$sisim", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name7", args{toByte("sisisisim$sisisisim", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name8", args{toByte("sisim1sisim", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name8", args{toByte("sisisisim$ananab", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name9", args{toByte("ananab$sisisisim", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name10", args{toByte("nana$abana$ananab$ananab", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name11", args{toByte("b$nab$aab", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name12", args{toByte("a1$a2$a3$b1$b2$b3$c1$c2$c3", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name13", args{toByte("ananabn$ananabn$ananab", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name13", args{toByte("anana$anana", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name14", args{toByte("atrt$snpsht$snpsht", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name15", args{toByte("atrt$snpshtsnpsht", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name15", args{toByte("snpshtsnpsht", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name16", args{toByte("OBu.:67 OBu.:35 OBu.:34", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name17", args{toByte("sisisim$sisisim$anana", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name18", args{toByte("0part$parent", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name19", args{toByte("reparent$parent", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name20", args{toByte("018-1$2", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
		{"name21", args{toByte("011$2", '$', 1), 10}, []byte{'a', 'a', 'n', 'b', 'n', 'n', 0, 'b', 'a', 1, 'a', 'a', 'a'}},
	}
	for _, tt := range tests {
		a := make([]byte, len(tt.args.t))
		copy(a, tt.args.t)
		t.Run(tt.name, func(t *testing.T) {
			if got, b, _ := BWT(tt.args.t); !reflect.DeepEqual(a, rewindBWT(b, got)) {
				t.Errorf("bwt() = %v, %v, want %v, %v", got, b, a, toString(rewindBWT(b, got), 1, '$'))
			}
		})
	}
}

func Test_sais(t *testing.T) {
	type args struct {
		t []byte
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		// mississippi
		{"0LMS", args{[]byte("abc")}, []int{0, 1, 2}},
		{"1LMS", args{[]byte("abcabca")}, []int{0, 3, 6, 1, 4, 2, 5}},
		{"name2", args{[]byte("ippississim")}, []int{0, 3, 6, 9, 10, 1, 2, 4, 7, 5, 8}},
		{"name3", args{[]byte("iippiissiissiimm")}, []int{0, 1, 5, 9, 13, 4, 8, 12, 14, 15, 2, 3, 6, 10, 7, 11}},
		{"name4", args{[]byte("sisisisim")}, []int{1, 3, 5, 7, 8, 0, 2, 4, 6}},
		{"name5", args{[]byte("sipisipisipisipisim")}, []int{3, 7, 11, 15, 1, 5, 9, 13, 17, 18, 2, 6, 10, 14, 0, 4, 8, 12, 16}},
		{"name6", args{toByte("sisisim$sisisim$anana", '$', 1)}, []int{7, 15, 16, 18, 20, 1, 9, 3, 11, 5, 13, 6, 14, 17, 19, 0, 8, 2, 10, 4, 12}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := make([]int, len(tt.args.t))
			if sais(bytebuf(tt.args.t), sa, 256, false, false); !reflect.DeepEqual(sa, tt.want) {
				t.Errorf("bwt() = %v, want %v", sa, tt.want)
			}
		})
	}
}

func Test_readfile(t *testing.T) {
	if datafile == "" {
		t.Skip("skiping readfile, -file option is empty")
	}
	d, err := ioutil.ReadFile(datafile)
	if err != nil {
		log.Fatal(err)
	}
	s := 0
	for _, b := range d {
		if b == '\n' {
			if s > 0 && d[s-1] != 1 {
				d[s] = 1
				s++
			}
			continue
		}
		d[s] = b
		s++
	}
	if d[s-1] == 1 {
		s--
	}
	d = d[:s]

	type args struct {
		t []byte
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"name2", args{d}, 134},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := make([]byte, len(tt.args.t))
			copy(a, tt.args.t)
			fmt.Printf("starting\n")
			s := time.Now()
			got, b, _ := BWT(tt.args.t)
			fmt.Printf("done in %v\n", time.Since(s))
			if !reflect.DeepEqual(a, rewindBWT(b, got)) {
				t.Errorf("bwt() = %v, %v, want %v", got, rewindBWT(b, got), len(a))
			} else {
				freeq := map[byte]int{}
				sz := 0
				for i, c := range b {
					if i > 0 && i%256 == 0 {
						t.Logf("%v\n", freeq)
						sz += len(freeq) * 2
						freeq = map[byte]int{}
					}
					freeq[c]++
				}
				if len(freeq) > 0 {
					t.Logf("%v\n", freeq)
				}
				sz += len(freeq) * 2
				t.Logf("%d size\n", sz)
			}
		})
	}
}
