package binio

// SetBit sets bit i within p to v, returning the new value of p.
func SetBit(p uint64, i int, v bool) uint64 {
	if v {
		return p | 1<<uint(i)
	}
	return p & ^(1 << uint(i))
}

// SetBits sets bits i through j within p to v, returning the new value of p.
func SetBits(p uint64, i, j, v int) uint64 {
	var m uint64 = 1<<uint(j-i) - 1
	return p&^(m<<uint(i)) | (uint64(v)&m)<<uint(i)
}

// GetBit gets bit i within p.
func GetBit(p uint64, i int) bool {
	return (p>>uint(i))&1 == 1
}

// GetBits gets bits i through j within p.
func GetBits(p uint64, i, j int) int {
	return int((p >> uint(i)) & (1<<uint(j-i) - 1))
}
