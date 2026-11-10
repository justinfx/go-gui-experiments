//go:build !cgoprobe

package core

import "hash/crc32"

// ChecksumUpdate extends a CRC-32 (IEEE) checksum with b. Export uses it on
// every row. With the cgoprobe build tag the same function is implemented
// in C through cgo.
func ChecksumUpdate(crc uint32, b []byte) uint32 { return crc32.Update(crc, crc32.IEEETable, b) }

// CgoProbe reports whether this build uses the cgo implementation.
const CgoProbe = false
