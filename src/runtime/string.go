// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"
)

func concatstrings(a []string) string {
	idx := 0
	l := 0
	count := 0
	for i, x := range a {
		n := len(x)
		if n == 0 {
			continue
		}
		if l+n < l {
			throw("string concatenation too long")
		}
		l += n
		count++
		idx = i
	}
	if count == 0 {
		return ""
	}
	if count == 1 {
		return a[idx]
	}
	s, b := rawstring(l)
	l = 0
	for _, x := range a {
		copy(b[l:], x)
		l += len(x)
	}
	return s
}

func concatstring2(a [2]string) string {
	return concatstrings(a[:])
}

func concatstring3(a [3]string) string {
	return concatstrings(a[:])
}

func concatstring4(a [4]string) string {
	return concatstrings(a[:])
}

func concatstring5(a [5]string) string {
	return concatstrings(a[:])
}

func slicebytetostring(b []byte) string {
	if raceenabled && len(b) > 0 {
		racereadrangepc(unsafe.Pointer(&b[0]),
			uintptr(len(b)),
			getcallerpc(unsafe.Pointer(&b)),
			funcPC(slicebytetostring))
	}
	s, c := rawstring(len(b))
	copy(c, b)
	return s
}

func slicebytetostringtmp(b []byte) string {
	// Return a "string" referring to the actual []byte bytes.
	// This is only for use by internal compiler optimizations
	// that know that the string form will be discarded before
	// the calling goroutine could possibly modify the original
	// slice or synchronize with another goroutine.
	// Today, the only such case is a m[string(k)] lookup where
	// m is a string-keyed map and k is a []byte.

	if raceenabled && len(b) > 0 {
		racereadrangepc(unsafe.Pointer(&b[0]),
			uintptr(len(b)),
			getcallerpc(unsafe.Pointer(&b)),
			funcPC(slicebytetostringtmp))
	}
	return *(*string)(unsafe.Pointer(&b))
}

func stringtoslicebyte(s string) []byte {
	b := rawbyteslice(len(s))
	copy(b, s)
	return b
}

func stringtoslicerune(s string) []rune {
	// two passes.
	// unlike slicerunetostring, no race because strings are immutable.
	n := 0
	t := s
	for len(s) > 0 {
		_, k := charntorune(s)
		s = s[k:]
		n++
	}
	a := rawruneslice(n)
	n = 0
	for len(t) > 0 {
		r, k := charntorune(t)
		t = t[k:]
		a[n] = r
		n++
	}
	return a
}

func slicerunetostring(a []rune) string {
	if raceenabled && len(a) > 0 {
		racereadrangepc(unsafe.Pointer(&a[0]),
			uintptr(len(a))*unsafe.Sizeof(a[0]),
			getcallerpc(unsafe.Pointer(&a)),
			funcPC(slicerunetostring))
	}
	var dum [4]byte
	size1 := 0
	for _, r := range a {
		size1 += runetochar(dum[:], r)
	}
	s, b := rawstring(size1 + 3)
	size2 := 0
	for _, r := range a {
		// check for race
		if size2 >= size1 {
			break
		}
		size2 += runetochar(b[size2:], r)
	}
	return s[:size2]
}

type stringStruct struct {
	str unsafe.Pointer
	len int
}

func intstring(v int64) string {
	s, b := rawstring(4)
	n := runetochar(b, rune(v))
	return s[:n]
}

// stringiter returns the index of the next
// rune after the rune that starts at s[k].
func stringiter(s string, k int) int {
	if k >= len(s) {
		// 0 is end of iteration
		return 0
	}

	c := s[k]
	if c < runeself {
		return k + 1
	}

	// multi-char rune
	_, n := charntorune(s[k:])
	return k + n
}

// stringiter2 returns the rune that starts at s[k]
// and the index where the next rune starts.
func stringiter2(s string, k int) (int, rune) {
	if k >= len(s) {
		// 0 is end of iteration
		return 0, 0
	}

	c := s[k]
	if c < runeself {
		return k + 1, rune(c)
	}

	// multi-char rune
	r, n := charntorune(s[k:])
	return k + n, r
}

// rawstring allocates storage for a new string. The returned
// string and byte slice both refer to the same storage.
// The storage is not zeroed. Callers should use
// b to set the string contents and then drop b.
func rawstring(size int) (s string, b []byte) {
	p := mallocgc(uintptr(size), nil, flagNoScan|flagNoZero)

	(*stringStruct)(unsafe.Pointer(&s)).str = p
	(*stringStruct)(unsafe.Pointer(&s)).len = size

	(*slice)(unsafe.Pointer(&b)).array = (*uint8)(p)
	(*slice)(unsafe.Pointer(&b)).len = uint(size)
	(*slice)(unsafe.Pointer(&b)).cap = uint(size)

	for {
		ms := maxstring
		if uintptr(size) <= uintptr(ms) || casuintptr((*uintptr)(unsafe.Pointer(&maxstring)), uintptr(ms), uintptr(size)) {
			return
		}
	}
}

// rawbyteslice allocates a new byte slice. The byte slice is not zeroed.
func rawbyteslice(size int) (b []byte) {
	cap := roundupsize(uintptr(size))
	p := mallocgc(cap, nil, flagNoScan|flagNoZero)
	if cap != uintptr(size) {
		memclr(add(p, uintptr(size)), cap-uintptr(size))
	}

	(*slice)(unsafe.Pointer(&b)).array = (*uint8)(p)
	(*slice)(unsafe.Pointer(&b)).len = uint(size)
	(*slice)(unsafe.Pointer(&b)).cap = uint(cap)
	return
}

// rawruneslice allocates a new rune slice. The rune slice is not zeroed.
func rawruneslice(size int) (b []rune) {
	if uintptr(size) > _MaxMem/4 {
		throw("out of memory")
	}
	mem := roundupsize(uintptr(size) * 4)
	p := mallocgc(mem, nil, flagNoScan|flagNoZero)
	if mem != uintptr(size)*4 {
		memclr(add(p, uintptr(size)*4), mem-uintptr(size)*4)
	}

	(*slice)(unsafe.Pointer(&b)).array = (*uint8)(p)
	(*slice)(unsafe.Pointer(&b)).len = uint(size)
	(*slice)(unsafe.Pointer(&b)).cap = uint(mem / 4)
	return
}

// used by cmd/cgo
func gobytes(p *byte, n int) []byte {
	if n == 0 {
		return make([]byte, 0)
	}
	x := make([]byte, n)
	memmove(unsafe.Pointer(&x[0]), unsafe.Pointer(p), uintptr(n))
	return x
}

func gostringsize(n int) string {
	s, _ := rawstring(n)
	return s
}

func gostring(p *byte) string {
	l := findnull(p)
	if l == 0 {
		return ""
	}
	s, b := rawstring(l)
	memmove(unsafe.Pointer(&b[0]), unsafe.Pointer(p), uintptr(l))
	return s
}

func gostringn(p *byte, l int) string {
	if l == 0 {
		return ""
	}
	s, b := rawstring(l)
	memmove(unsafe.Pointer(&b[0]), unsafe.Pointer(p), uintptr(l))
	return s
}

func index(s, t string) int {
	if len(t) == 0 {
		return 0
	}
	for i := 0; i < len(s); i++ {
		if s[i] == t[0] && hasprefix(s[i:], t) {
			return i
		}
	}
	return -1
}

func contains(s, t string) bool {
	return index(s, t) >= 0
}

func hasprefix(s, t string) bool {
	return len(s) >= len(t) && s[:len(t)] == t
}

func atoi(s string) int {
	n := 0
	for len(s) > 0 && '0' <= s[0] && s[0] <= '9' {
		n = n*10 + int(s[0]) - '0'
		s = s[1:]
	}
	return n
}
