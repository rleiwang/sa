/*
 * Copyright 2020 Rock Lei Wang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Package parser declares an expression parser with support for macro
 * expansion.
 */

package sa

const (
	alphabetSize = 256
	separator    = 1
)

// Aux Auxiliary for merge
type Aux struct {
	// Len length
	Len uint

	// Eob End of Bucket
	Eob [][]uint

	// Dist distribution
	Dist []uint

	// Hist histogram
	Hist []uint

	// dictionary, [0] -> 0, [1] -> 1, [2] -> can be any char
	Dict []byte
}

type buf interface {
	len() int
	get(i int) int
	eq(x, y int) bool
}

type bytebuf []byte
type intbuf []int

// start bytebuf

func (b bytebuf) len() int {
	return len(b)
}

func (b bytebuf) get(i int) int {
	return int(b[i])
}

func (b bytebuf) eq(x, y int) bool {
	return b[x] == b[y]
}

// end bytebuf

// start intbuf

func (b intbuf) len() int {
	return len(b)
}

func (b intbuf) get(i int) int {
	return int(b[i])
}

func (b intbuf) eq(x, y int) bool {
	return b[x] == b[y]
}

// end intbuf

func reset(chars []uint) {
	chars[0] = 0
	sz := 1
	for sz < 256 {
		copy(chars[sz:], chars[:sz])
		sz <<= 1
	}
}

// BWT transforms t into BWT, returns the length of BWT, BWT and auxiliary data structure can be used to merge BWTs
func BWT(t []byte) (int, []byte, *Aux) {
	sa := make([]int, len(t))
	// dict -> 0 -> 0, 1 -> 1, 2 -> '\n'
	l, arr, dict := sais(bytebuf(t), sa, alphabetSize, true, false)
	t = append(t, 1)

	// note: dict content is ascending, make sure byte 0 and byte 1 are indexed 0 and 1
	if dict[0] != 0 {
		dict = append([]byte{0, 1}, dict...)
	} else if dict[1] != 1 {
		dict = append([]byte{0, 1}, dict[1:]...)
	}

	// note: Dist starts with one ZERO value
	aux := &Aux{uint(len(t)), make([][]uint, 256, 256), []uint{0}, []uint{}, dict}
	for i := range aux.Eob {
		aux.Eob[i], arr = arr[:256], arr[256:]
	}

	sum, chars := uint(0), make([]uint, 256, 256)
	for _, rnk := range aux.Eob {
		for j, r := range rnk {
			if r > 0 {
				reset(chars)
				sum += r
				for bi := sum - r; bi < sum; bi++ {
					if bi == 0 {
						chars[t[0]]++
					} else {
						t[bi] = byte(sa[bi-1])
						if t[bi] == 0 {
							chars[1]++
						} else {
							chars[t[bi]]++
						}
					}
				}
				for k, cnt := range chars {
					if cnt > 0 {
						aux.Hist = append(aux.Hist, (cnt<<8)|uint(k))
					}
				}

				rnk[j] = uint(len(aux.Dist))
				aux.Dist = append(aux.Dist, uint(len(aux.Hist)))
			}
		}
	}

	return l + 1, t, aux
}

// text, sa, alphabet size, output as bwt, recursive
func sais(t buf, sa []int, k int, bwt, rec bool) (int, []uint, []byte) {
	// scan text to create distribution histgram
	hist, bkt := histgram(t, k)

	// m -> number of LMS excluding sentinel
	// ms -> number of separators
	// ┌0────5────0───-5────0┐
	// │sisisim$sisisim$anana│
	// └─*─*─*─#─*─*-*─#──*──┘
	// m = 9, ms = 2
	m, ms := findLMS(t, sa, bkt, hist, rec)
	if m > 1 || ms > 1 {
		// inducing sort LMS substrings into their relative positions, including separators, except sentinel
		// ┌0─┬───┬──┬──┬───┬5─┬───┬───┬───┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
		// │-8│-16│  │  │-19│-2│-10│-14│-12│-6│-4│  │  │  │  │  │  │  │  │  │  │
		// └▲─┴───┼──┴──┴───┼──┴───┴───┴───┴──┴▲─┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
		//       sep        a                    i     m     n                 s
		sortLMS(t, sa, bkt, hist, ms)

		// name LMS substrings in lexicographic order, n -> number of LMS with unique name
		//        ┌────────────────────────────────────────────────┐
		// ┌0─┬──┬┴─┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬─▼┬──┬20┐
		// │ 7│15│18│ 1│ 9│13│11│ 5│ 3│ 4│ 6│ 6│ 1│ 5│ 6│ 6│ 2│  │ 3│  │  │
		// └──┴──┴──┴──┴──┴──┴──┴──┴──▲─-┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
		//                            │m
		n := nameLMS(t, sa, m, rec)

		if n < m {
			// there are more than one LMS strings with the same lexicographical order

			// adjust LMS names and pass to recursive sais, where sa[m:2m] is T[...]
			// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
			// │  │  │  │  │  │  │  │  │  │ 3│ 5│ 5│ 0│ 4│ 5│ 5│ 1│ 2│  │  │  │
			// └──┴──┴──┴──┴──┴──┴──┴──┴──▲──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
			//                            │m
			adjustLMS(sa, m)

			// after recursive sais, sa[:m] returns suffix array of int names in sa[m:2m]
			// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
			// │ 3│ 7│ 8│ 0│ 4│ 1│ 5│ 2│ 6│ 3│ 5│ 5│ 0│ 4│ 5│ 5│ 1│ 2│  │  │  │
			// └──┴──┴──┴──┴──┴──┴──┴──┴──▲──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
			//                            │m
			sais(intbuf(sa[m:2*m]), sa[:m], n+1, false, true)

			// locate and shuffle LMS into lexicographic order in sa[:m]
			// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
			// │ 7│15│18│ 1│ 9│ 3│11│ 5│13│  │  │  │  │  │  │  │  │  │  │  │  │
			// └──┴──┴──┴──┴──┴──┴──┴──┴──▲──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
			//                            │m
			locateLMS(t, sa, m, rec)
			shuffleLMS(sa, m)
		} else {
			// reset sa[m:] to zero
			clearLMSLen(sa, m)
		}

		// restore LMS into their corresponding buckets
		// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
		// │8 │16│  │  │19│ 2│10│ 4│12│ 6│14│  │  │  │  │  │  │  │  │  │  │
		// └──┴──┼──┴──┴──┼──┴──┴──┴──┴──┴──┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
		//      sep       a                 i     m     n                 s
		restoreLMS(t, sa, bkt, hist, m)
	}

	if bwt {
		return induceBWT(t, sa, bkt, hist, ms)
	}
	return induce(t, sa, bkt, hist, ms), nil, nil
}

func induce(t buf, sa, bkt, hist []int, ms int) int {
	setBktBeg(bkt, hist)

	// sentinel is LMS, T[0] is L type
	// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │8 │16│  │  │19│ 2│10│ 4│12│ 6│14│  │  │  │  │-1│  │  │  │  │  │
	// └──┴──┼──┴──┴──┼──┴──┴──┴──┴──┴──┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//      sep       a                 i     m     n                 s
	n, p, end := 0, t.get(0), t.len()
	b := bkt[p]
	if p > t.get(1) {
		// next suffix is S type, put ^sa[i]
		sa[b] = ^0
	} else {
		// next suffix is L type, put sa[i] + 1
		sa[b] = 1
	}
	b++

	// scan sa from left to right to induce L type
	// ┌0─┬───┬───┬──┬───┬5─┬───┬──┬───┬──┬─10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │-8│-16│-17│  │-19│-2│-10│-4│-12│-6│-14│ 6│14│17│19│ 0│ 8│ 2│10│ 4│12│
	// └──┴───┼───┴──┴───┼──┴───┴──┴───┴──┴───┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//       sep         a                    i     m     n                 s
	for i, s := range sa {
		if s == 0 {
			// skip if s == 0
			continue
		} else if s < 0 {
			// S type, to induce later
			sa[i] = ^s
			continue
		}

		if s >= end {
			// reached end of buffer, note: inducing L type, sa contains sa[i] + 1
			// store negative number, no need to induce S
			sa[i] = ^(s - 1)
			continue
		}

		sa[i] = ^(s - 1)

		n = t.get(s)
		if p != n {
			bkt[p], b, p = b, bkt[n], n
		}

		if s+1 < end && p > t.get(s+1) {
			// S type
			sa[b] = ^s
		} else {
			// L type
			sa[b] = s + 1
		}
		b++
	}

	// scan sa from right to left to induce S type, note: sa contains sa[i] for all S type
	// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │ 7│15│16│18│20│ 1│ 9│ 3│11│ 5│13│ 6│14│17│19│ 0│ 8│ 2│10│ 4│12│
	// └──┴──┼──┴──┴──┼──┴──┴──┴──┴──┴──┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//      sep       a                 i     m     n                 s
	setBktEnd(bkt, hist)
	p, b = 0, bkt[0]
	for i := end - 1; i >= 0; i-- {
		s := sa[i]
		if s < 0 {
			// sorted L type substring
			sa[i] = ^s
			continue
		}

		s++
		if s == end {
			// reached end of buffer
			continue
		}
		n = t.get(s)
		if p != n {
			bkt[p], b, p = b, bkt[n], n
		}
		b--
		if s == end-1 || p < t.get(s+1) {
			// next suffix is L type
			sa[b] = ^s
		} else {
			// S type suffix
			sa[b] = s
		}
	}
	return -1
}

func updateRank(rank [][]uint, a, b int) {
	if a < 1 {
		a = 1
	}

	rank[a][b]++
}

// same as induce except, it produces BWT and data structure for merging BWT
func induceBWT(t buf, sa, bkt, hist []int, ms int) (int, []uint, []byte) {
	cnt := countBktBeg(bkt, hist)
	ptr, dict, blk, rnk := makeCounters(bkt, hist, cnt)

	// sentinel is LMS, T[0] is L type
	// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │8 │16│  │  │19│ 2│10│ 4│12│ 6│14│  │  │  │  │-1│  │  │  │  │  │
	// └──┴──┼──┴──┴──┼──┴──┴──┴──┴──┴──┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//      sep       a                 i     m     n                 s
	n, p, end := 0, t.get(0), t.len()
	b := bkt[p]
	if t.len() > 1 && p > t.get(1) {
		// next suffix is S type, put ^sa[i]
		sa[b] = ^0
	} else {
		// next suffix is L type, put sa[i] + 1
		sa[b] = 1
	}
	b++
	// rnk[p][1]++
	updateRank(rnk, p, 1)

	// scan sa from left to right to induce L type
	// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │-s│-a│-n│  │-n│-s│-s│-s│-s│-m│-m│ 6│14│17│19│ 0│ 8│ 2│10│ 4│12│
	// └──┴──┼──┴──┴──┼──┴──┴──┴──┴──┴──┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//      sep       a                 i     m     n                 s
	for i, s := range sa {
		if s == 0 {
			// skip if s == 0
			continue
		} else if s < 0 {
			// S type, to induce later
			sa[i] = ^s
			continue
		}

		if s >= end {
			// reached end of buffer, note: inducing L type, sa contains sa[i] + 1
			// it's end of buffer, sa[i] will assign to 0 when inducing S type suffix
			sa[i] = end - 1
			continue
		}

		n = t.get(s)
		// BWT T[sa[i]+1]
		sa[i] = ^n

		if p != n {
			bkt[p], b, p = b, bkt[n], n
		}

		if s+1 < end && p > t.get(s+1) {
			// S type
			sa[b] = ^s
		} else {
			// L type
			sa[b] = s + 1
		}
		b++
	}

	// scan sa from right to left to induce S type, note: sa contains sa[i] for all S type
	// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │ s│ a│ n│ n│ 0│ s│ s│ s│ s│ m│ m│ 1│ 1│ a│ a│ i│ i│ i│ i│ i│ i│
	// └──┴──┼──┴──┴──┼──┴──┴──┴──┴──┴──┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//      sep       a                 i     m     n                 s
	setBktEnd(bkt, hist)
	p, b = 0, bkt[0]
	l, idx := 0, cnt-1
	for i := end - 1; i >= 0; i-- {
		if i < ptr[idx] {
			idx--
		}
		s := sa[i]
		if s < 0 {
			// sorted L type substring
			sa[i] = ^s
			// rnk[sa[i]][dict[idx]]++
			updateRank(rnk, sa[i], int(dict[idx]))
			continue
		}

		s++
		if s == end {
			// reached end of buffer
			l = i
			sa[i] = 0
			updateRank(rnk, 1, int(dict[idx]))
			continue
		}
		n = t.get(s)
		sa[i] = n
		// rnk[n][dict[idx]]++
		updateRank(rnk, n, int(dict[idx]))

		if p != n {
			bkt[p], b, p = b, bkt[n], n
		}
		b--
		if s == end-1 || p < t.get(s+1) {
			// next suffix is L type
			if s == end-1 {
				// must not have separator at the end of buffer
				l = b
				sa[b] = ^0
			} else if b >= ms {
				// only if sa[b] is not separator, bwt is T[sa[i] + 1]
				sa[b] = ^t.get(s + 1)
			}
		} else {
			// S type suffix
			sa[b] = s
		}
	}
	return l, blk, dict
}

// place named LMS substrings to sa[m:2m]
func adjustLMS(sa []int, m int) {
	// reset sa[:m]
	for i := 0; i < m; i++ {
		sa[i] = 0
	}
	// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │  │  │  │  │  │  │  │  │  │ 3│ 5│ 5│ 0│ 4│ 5│ 5│ 1│ 2│  │  │  │
	// └──┴──┴──┴──┴──┴──┴──┴──┴──▲──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
	//                            │m
	for i, j, end := m, 0, len(sa); i < end && j < m; i++ {
		if sa[i] > 0 {
			// sa[i] contains sa[j] + 1 (ie, L suffix offset of LMS)
			if m+j != i {
				sa[j+m] = sa[i] - 1
				sa[i] = 0
			} else {
				sa[i]--
			}
			j++
		}
	}
}

// scan T to locate LMS and put their offset in sa[m:2m]
// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
// │ 3│ 7│ 8│ 0│ 4│ 1│ 5│ 2│ 6│ 1│ 3│ 5│ 7│ 9│11│13│15│18│  │  │  │
// └──┴──┴──┴──┴──┴──┴──┴──┴──▲──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
//                            │m
func locateLMS(t buf, sa []int, m int, rec bool) {
	p, s := t.get(0), false
	for i, e := 1, t.len(); i < e; i++ {
		c := t.get(i)
		if s {
			if p < c {
				sa[m], s = i-1, false
				m++
			}
		} else if p > c {
			s = true
		}
		p = c
	}
}

// shuffle LMS suffix from sa[m:2m] to sa[:m]
func shuffleLMS(sa []int, m int) {
	// sa[:m] contains suffix array of LMS suffix in sa[m:2m]
	// loop sa[:m], put LMS suffix to sa[:m] in their lexicographical order
	for i := 0; i < m; i++ {
		j := m + sa[i]
		sa[i], sa[j] = sa[j], 0
	}
}

func clearLMSLen(sa []int, m int) {
	for i := m - 1; i >= 0; i-- {
		sa[m+sa[i]>>1] = 0
	}
}

// put L suffix's offset of LMS suffix into position from end their corresponding buckets
// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
// │8 │16│  │  │19│ 2│10│ 4│12│ 6│14│  │  │  │  │  │  │  │  │  │  │
// └──┴──┼──┴──┴──┼──┴──┴──┴──┴──┴──┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
//      sep       a                 i     m     n                 s
func restoreLMS(t buf, sa, bkt, hist []int, m int) {
	setBktEnd(bkt, hist)
	p, b := 0, bkt[0]
	for i := m - 1; i >= 0; i-- {
		c := t.get(sa[i])
		if p != c {
			bkt[p], b, p = b, bkt[c], c
		}
		b--
		if b != i {
			sa[b], sa[i] = sa[i]+1, 0
		} else {
			sa[i]++
		}
	}
}

// return number of LMS excluding sentinel
// ┌0────5────0───-5────0┐
// │sisisim$sisisim$anana│
// └─*─*─*─#─*─*-*─#──*──┘
// Separators have builtin ascending lexicographic order. ie $[7] < $[15] in above example
// No need to sort separators. Place separators from start of the bucket to sort LMS as sentinel.
// Induce sort LMS into its relative positions. Note: SA contains the offset of L suffix for LMS.
// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
// │8 │16│  │  │  │  │12│10│  │4 │2 │  │  │  │  │  │  │  │  │  │  │
// └▲─┴──┼──┴──┴──┼──┴──┴──┴──┴──┴▲─┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
//      sep       a                 i     m     n                 s
func findLMS(t buf, sa, bkt, hist []int, rec bool) (int, int) {
	setBktEnd(bkt, hist)
	m, ms, l, lms, sbkt := 0, 0, 0, -1, bkt[separator-1]
	p := t.get(0)
	s := false
	for i, e := 1, t.len(); i < e; i++ {
		// p -> t[i - 1], c -> t[i]
		c := t.get(i)
		// TODO: check p == c == separator, separator is LMS,
		// for two separators, one is S, the next one is LMS
		if s && p < c {
			// p -> LMS, m -> number of LMS
			m++
			if !rec && p == separator {
				// separators lexicographic ordered, no need to sort,
				// put separators in place from start of the bucket
				sa[sbkt], lms = i, -1
				sbkt++
				ms++
			} else {
				if lms >= 0 {
					// sentinel and separators are sorted at the beginning of the array
					sa[lms] = l
				}
				bkt[p]--
				lms, l = bkt[p], i
			}
			s = false
		} else if p > c {
			// if s true, p == c, then, c is S as well
			s = true
		}
		p = c
	}
	if m == 1 && lms >= 0 {
		// note: l is the L suffix to the right of LMS
		// add L of the sentinel, will go induce directly
		// if there is only one LMS and it is separator, m == 1, lms == -1,
		sa[lms] = l
	}
	return m, ms
}

func nameLMS(t buf, sa []int, m int, rec bool) int {
	// compact all the sorted substrings into the first m items of SA
	// 2*m must be not larger than n (proveable)
	// ┌0─┬──┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │ 7│15│18│ 1│ 9│13│11│ 5│ 3│  │  │  │  │  │  │  │  │  │  │  │  │
	// └──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
	j := 0
	for i, s := range sa {
		if s < 0 {
			sa[j] = ^s
			if j < i {
				sa[i] = 0
			}
			j++
			if j == m {
				break
			}
		}
	}

	// search text for LMS and place length of LMS substring at sa[m + i/2]
	//        ┌────────────────────────────────────────────────┐
	// ┌0─┬──┬┴─┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬─▼┬──┬20┐
	// │ 7│15│18│ 1│ 9│13│11│ 5│ 3│ 2│ 3│ 3│-1│ 3│ 3│ 3│-1│  │ 4│  │  │
	// └──┴──┴──┴──┴──┴──┴──┴──┴──▲──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
	//                            │m
	// note: sa[3] is 18, it's length is at sa[m + 18 >> 1], ie sa[9 + 9]
	p, j, s := t.get(0), 0, false
	for i, e := 1, t.len(); i < e; i++ {
		n := t.get(i)
		if s && p < n {
			// L type, LMS is last suffix, +1 sentinel
			if !rec && p == separator {
				sa[m+((i-1)>>1)] = -1
			} else {
				sa[m+((i-1)>>1)] = i - j
			}
			s, j = false, i-1
		} else if p > n {
			// if S true, n == p then, n is
			s = true
		}
		p = n
	}

	// n -> names, starts names with one, no need to name/sort sentinel
	// separators' length is minus one, must be different
	//        ┌────────────────────────────────────────────────┐
	// ┌0─┬──┬┴─┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬─▼┬──┬20┐
	// │ 7│15│18│ 1│ 9│13│11│ 5│ 3│ 4│ 6│ 6│ 1│ 5│ 6│ 6│ 2│  │ 3│  │  │
	// └──┴──┴──┴──┴──┴──┴──┴──┴──▲─-┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┘
	//                            │m
	plen, b, n := -1, sa[0]>>1, 1
	plen, sa[m+b] = sa[m+b], n
	for i := 1; i < m; i++ {
		b = sa[i] >> 1
		diff := true
		if plen > 0 && plen == sa[m+b] {
			// two LMS suffix with same length and not separators
			x, y, l := sa[i-1], sa[i], 0
			for l < plen && x >= 0 && y >= 0 && t.eq(x, y) {
				l++
				x--
				y--
			}
			// one char in two suffix is different if l < plen
			diff = l < plen
			if !rec && !diff && x >= 0 && y >= 0 && t.get(x) == separator && t.get(y) == separator {
				// if two LMS substrings are equal but ends with separators, they should be different
				diff = true
			}
		} else {
			// two suffix with different length, must be different
			plen = sa[m+b]
		}
		if diff {
			n++
		}
		sa[m+b] = n
	}

	return n
}

// sort LMS, note: LMS in sa[] contains L suffix
func sortLMS(t buf, sa, bkt, hist []int, ms int) {
	setBktBeg(bkt, hist)
	// T[0] is L type suffix, set sa[0]
	// moving forward store next char b in sa
	// p -> left char, b -> current bucket index

	// sort sentinel, T[0] is always L suffix because sentinel is LMS
	p, sz := t.get(0), t.len()
	b := bkt[p]
	if p > t.get(1) {
		// T[1] is S
		sa[b] = ^1
	} else {
		// T[1] is L
		sa[b] = 1
	}
	b++

	// sort L type
	// ┌0─┬───┬──┬──┬──┬5─┬──┬──┬──┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │-8│-16│  │  │  │  │  │  │  │  │  │  │  │18│  │ 1│ 9│13│11│ 5│ 3│
	// └▲─┴───┼──┴──┴──┼──┴──┴──┴──┴──┴▲─┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//       sep       a                 i     m     n                 s
	for i, s := range sa {
		if s > 0 {
			// sa[i] > 0 is L suffix, sorted sentinel at the start of this function
			// note: when sorting LMS sa[i] = sa[j] + 1, where j is the offset of either LMS or L
			c := t.get(s)
			if p != c {
				// different char
				bkt[p], b, p = b, bkt[c], c
			}
			s++
			if s >= sz {
				// reached end of buffer, there is no LMS suffix, stop
				sa[b] = 0
			} else if p > t.get(s) {
				// next suffix is S type
				sa[b] = ^s
			} else {
				// next suffix is L type
				sa[b] = s
			}
			b++
			if i >= ms {
				sa[i] = 0
			} else {
				// separators are at the start of SA, they are sorted
				// note: sa[i] is offset of the L suffix for all LMS
				sa[i] = ^(sa[i] - 1)
			}
		} else if s < 0 {
			// XOR negative to positive, sa[i] < 0 if sa[i] is S
			sa[i] = ^s
		}
	}

	// sort S type
	// ┌0─┬───┬──┬──┬───┬5─┬───┬───┬───┬──┬10┬──┬──┬──┬──┬15┬──┬──┬──┬──┬20┐
	// │-8│-16│  │  │-19│-2│-10│-14│-12│-6│-4│  │  │  │  │  │  │  │  │  │  │
	// └▲─┴───┼──┴──┴───┼──┴───┴───┴───┴──┴▲─┼──┴──┼──┴──┼──┴──┴──┴──┴──┴──┤
	//       sep        a                    i     m     n                 s
	setBktEnd(bkt, hist)
	b, p = bkt[0], 0
	for i := len(sa) - 1; i >= 0; i-- {
		s := sa[i]
		if s > 0 {
			// sa[i] > 0 are S suffix, L suffix sa[i] == 0, separators sa[i] < 0
			c := t.get(s)
			if p != c {
				// different char
				bkt[p], b, p = b, bkt[c], c
			}
			b--
			s++
			if s >= sz {
				// reached end of buffer, there is no LMS suffix, stop
				sa[b] = 0
			} else if p < t.get(s) {
				// next suffix is L type
				if b >= ms {
					// LMS suffix, no need if b < ms, separators are sorted
					// note: sa[i], ie s, contains offset of L suffix, ie sa[j] + 1
					sa[b] = ^(s - 1)
				}
			} else {
				// next suffix is S type
				sa[b] = s
			}
			sa[i] = 0
		}
	}
}

func histgram(t buf, k int) ([]int, []int) {
	h := make([]int, k)

	for i, end := 0, t.len(); i < end; i++ {
		h[t.get(i)]++
	}
	return h, make([]int, k)
}

func setBktEnd(bkt, hist []int) {
	sum := int(0)
	for i, h := range hist {
		sum += h
		bkt[i] = sum
	}
}

func setBktBeg(bkt, hist []int) {
	sum := int(0)
	for i, h := range hist {
		bkt[i] = sum
		sum += h
	}
}

func countBktBeg(bkt, hist []int) int {
	sum := int(0)
	cnt := 1
	for i, h := range hist {
		if h > 0 && i > 1 {
			cnt++
		}
		bkt[i] = sum
		sum += h
	}

	return cnt
}

func makeCounters(bkt, hist []int, cnt int) ([]int, []byte, []uint, [][]uint) {
	lvl := make([]int, cnt, cnt)
	dict := make([]byte, cnt, cnt)
	arr := make([]uint, 256*256, 256*256)
	pops := make2Darr(arr, 256)

	idx := cnt - 1
	for i := byte(255); i > 1; i-- {
		if hist[i] > 0 {
			dict[idx] = i
			lvl[idx] = bkt[i]
			idx--
		}
	}

	return lvl, dict, arr, pops
}

func make2Darr(s []uint, sz int) [][]uint {
	arr := make([][]uint, sz, sz)

	for i := range arr {
		arr[i], s = s[:sz], s[sz:]
	}

	return arr
}
