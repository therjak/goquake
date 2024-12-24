// SPDX-License-Identifier: GPL-2.0-or-later

//go:generate  protoc -I/usr/local/include -I. --go_out=. --go_opt=paths=source_relative savegame.proto
//go:generate  protoc -I/usr/local/include -I. --go_out=. --go_opt=paths=source_relative history.proto
//go:generate  protoc -I/usr/local/include -I. --go_out=. --go_opt=paths=source_relative client_message.proto
//go:generate  protoc -I/usr/local/include -I. --go_out=. --go_opt=paths=source_relative server_message.proto
package protos
