// SPDX-License-Identifier: GPL-2.0-or-later

package protocol

const (
	NetQuake  = 15
	FitzQuake = 666
	RMQ       = 999
	GoQuake   = 9090
)

const (
	PRFL_SHORTANGLE = (1 << 1)
	PRFL_FLOATANGLE = (1 << 2)
	PRFL_24BITCOORD = (1 << 3)
	PRFL_FLOATCOORD = (1 << 4)
	PRFL_EDICTSCALE = (1 << 5)
	PRFL_INT32COORD = (1 << 7)
)

const (
	MaxDatagram = 32000
)

const (
	ANGLESHORT  = 1 << 1
	ANGLEFLOAT  = 1 << 2
	COORD24BIT  = 1 << 3
	COORDFLOAT  = 1 << 4
	EDICTSCALE  = 1 << 5
	ALPHASANITY = 1 << 6
	COORDINT32  = 1 << 7
	MOREFLAGS   = 1 << 31 // not supported
)
