//go:build cgoprobe

package core

/*
#include <stdint.h>
#include <stddef.h>

static uint32_t gguix_crc32_update(uint32_t crc, const unsigned char *p, size_t n) {
	crc = ~crc;
	for (size_t i = 0; i < n; i++) {
		crc ^= p[i];
		for (int k = 0; k < 8; k++) {
			crc = (crc >> 1) ^ (0xEDB88320u & (0u - (crc & 1u)));
		}
	}
	return ~crc;
}
*/
import "C"

import "unsafe"

// ChecksumUpdate extends a CRC-32 (IEEE) checksum with b, implemented in C.
func ChecksumUpdate(crc uint32, b []byte) uint32 {
	if len(b) == 0 {
		return crc
	}
	return uint32(C.gguix_crc32_update(C.uint32_t(crc), (*C.uchar)(unsafe.Pointer(&b[0])), C.size_t(len(b))))
}

// CgoProbe reports whether this build uses the cgo implementation.
const CgoProbe = true
