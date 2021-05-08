// SPDX-License-Identifier: GPL-2.0-or-later

package keys

type Destination byte

const (
	Game = Destination(iota)
	Console
	Message
	Menu
)
