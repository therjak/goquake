// SPDX-License-Identifier: GPL-2.0-or-later
//go:generate  protoc --go_out=. savegame.proto
//go:generate  protoc --go_out=. history.proto
//go:generate  protoc --go_out=. client_message.proto
//go:generate  protoc --go_out=. server_message.proto
package protos
