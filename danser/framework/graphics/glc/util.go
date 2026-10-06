package glc

import "unsafe"

func unsafeSlice(p *uint32, n int32) []uint32 {
	return unsafe.Slice(p, int(n))
}
