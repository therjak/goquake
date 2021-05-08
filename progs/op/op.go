// SPDX-License-Identifier: GPL-2.0-or-later

package op

const (
	DONE = iota
	MUL_F
	MUL_V
	MUL_FV
	MUL_VF
	DIV_F
	ADD_F
	ADD_V
	SUB_F
	SUB_V

	EQ_F
	EQ_V
	EQ_S
	EQ_E
	EQ_FNC

	NE_F
	NE_V
	NE_S
	NE_E
	NE_FNC

	LE
	GE
	LT
	GT

	LOAD_F
	LOAD_V
	LOAD_S
	LOAD_ENT
	LOAD_FLD
	LOAD_FNC

	ADDRESS

	STORE_F
	STORE_V
	STORE_S
	STORE_ENT
	STORE_FLD
	STORE_FNC

	STOREP_F
	STOREP_V
	STOREP_S
	STOREP_ENT
	STOREP_FLD
	STOREP_FNC

	RETURN
	NOT_F
	NOT_V
	NOT_S
	NOT_ENT
	NOT_FNC
	IF
	IFNOT
	CALL0
	CALL1
	CALL2
	CALL3
	CALL4
	CALL5
	CALL6
	CALL7
	CALL8
	STATE
	GOTO
	AND
	OR

	BITAND
	BITOR
)
