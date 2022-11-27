// SPDX-License-Identifier: GPL-2.0-or-later

// package input handles button event tracking
package input

import (
	"goquake/cmd"
)

type button struct {
	// key nums holding it down, can handle 2 keys with the same action
	holdingDown [2]int
	down        bool
	impulseDown bool
	impulseUp   bool
}

var (
	MLook     button
	KLook     button
	Left      button
	Right     button
	Forward   button
	Back      button
	LookUp    button
	LookDown  button
	MoveLeft  button
	MoveRight button
	Strafe    button
	Speed     button
	Use       button
	Jump      button
	Attack    button
	Up        button
	Down      button
)

func (b button) Down() bool {
	return b.down
}

func (b *button) WentDown() bool {
	// return down + impulse down
	// reset impulse down
	r := b.down || b.impulseDown
	b.impulseDown = false
	return r
}

// Returns 0.25 if a button was pressed and released during the frame,
// 0.5 if it was pressed and held
// 0 if held then released, and
// 1 if held for the entire time
func (b button) GetImpulse() float32 {
	if b.impulseDown && b.impulseUp {
		if b.down {
			return 0.75
		}
		return 0.25
	}
	if !b.impulseDown && !b.impulseUp {
		if b.down {
			return 1
		}
		return 0
	}
	if b.impulseUp && !b.impulseDown {
		return 0
	}
	if b.impulseDown && !b.impulseUp {
		if b.down {
			return 0.5
		}
		return 0
	}
	return 0 // unreachable
}

func (b *button) ResetImpulse() {
	b.impulseDown = false
	b.impulseUp = false
}

func (b *button) ConsumeImpulse() float32 {
	i := b.GetImpulse()
	b.ResetImpulse()
	return i
}

func (b *button) upKey(k int) {
	if b.holdingDown[0] == k {
		b.holdingDown[0] = 0
	} else if b.holdingDown[1] == k {
		b.holdingDown[1] = 0
	} else {
		return
	}
	if b.holdingDown[0] != 0 || b.holdingDown[1] != 0 {
		// some other key is still holding it down
		return
	}
	if !b.down {
		return
	}
	b.down = false
	b.impulseUp = true
}

func (b *button) upCmd(a []cmd.QArg) {
	if len(a) == 0 {
		// typed manually
		b.holdingDown[0] = 0
		b.holdingDown[1] = 0
		b.down = false
		b.impulseDown = false
		b.impulseUp = true
	} else {
		b.upKey(a[0].Int())
	}
}

func (b *button) downKey(k int) {
	if b.holdingDown[0] == 0 {
		b.holdingDown[0] = k
	} else if b.holdingDown[1] == 0 {
		b.holdingDown[1] = k
	} else {
		// Con_Printf("three key down for a button!\n")
		return
	}
	if b.down {
		return
	}
	b.down = true
	b.impulseDown = true
}

func (b *button) downCmd(a []cmd.QArg) {
	if len(a) == 0 {
		// typed manually
		b.downKey(-1)
	} else {
		b.downKey(a[0].Int())
	}
}

func init() {
	// Key events issue these commands and pass the key number as argument,
	// if no number expect console/cfg input
	cmd.Must(cmd.AddCommand("+moveup", func(a cmd.Arguments, p, s int) error {
		Up.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-moveup", func(a cmd.Arguments, p, s int) error {
		Up.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+movedown", func(a cmd.Arguments, p, s int) error {
		Down.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-movedown", func(a cmd.Arguments, p, s int) error {
		Down.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+left", func(a cmd.Arguments, p, s int) error {
		Left.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-left", func(a cmd.Arguments, p, s int) error {
		Left.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+right", func(a cmd.Arguments, p, s int) error {
		Right.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-right", func(a cmd.Arguments, p, s int) error {
		Right.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+forward", func(a cmd.Arguments, p, s int) error {
		Forward.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-forward", func(a cmd.Arguments, p, s int) error {
		Forward.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+back", func(a cmd.Arguments, p, s int) error {
		Back.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-back", func(a cmd.Arguments, p, s int) error {
		Back.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+lookup", func(a cmd.Arguments, p, s int) error {
		LookUp.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-lookup", func(a cmd.Arguments, p, s int) error {
		LookUp.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+lookdown", func(a cmd.Arguments, p, s int) error {
		LookDown.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-lookdown", func(a cmd.Arguments, p, s int) error {
		LookDown.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+strafe", func(a cmd.Arguments, p, s int) error {
		Strafe.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-strafe", func(a cmd.Arguments, p, s int) error {
		Strafe.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+moveleft", func(a cmd.Arguments, p, s int) error {
		MoveLeft.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-moveleft", func(a cmd.Arguments, p, s int) error {
		MoveLeft.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+moveright", func(a cmd.Arguments, p, s int) error {
		MoveRight.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-moveright", func(a cmd.Arguments, p, s int) error {
		MoveRight.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+speed", func(a cmd.Arguments, p, s int) error {
		Speed.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-speed", func(a cmd.Arguments, p, s int) error {
		Speed.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+attack", func(a cmd.Arguments, p, s int) error {
		Attack.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-attack", func(a cmd.Arguments, p, s int) error {
		Attack.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+use", func(a cmd.Arguments, p, s int) error {
		Use.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-use", func(a cmd.Arguments, p, s int) error {
		Use.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+jump", func(a cmd.Arguments, p, s int) error {
		Jump.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-jump", func(a cmd.Arguments, p, s int) error {
		Jump.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+klook", func(a cmd.Arguments, p, s int) error {
		KLook.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-klook", func(a cmd.Arguments, p, s int) error {
		KLook.upCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("+mlook", func(a cmd.Arguments, p, s int) error {
		MLook.downCmd(a.Args()[1:])
		return nil
	}))
	cmd.Must(cmd.AddCommand("-mlook", func(a cmd.Arguments, p, s int) error {
		MLook.upCmd(a.Args()[1:])
		// if !MLook.down && Cvar_GetValue(&lookspring) {
		// V_StartPitchDrift()
		// }
		return nil
	}))
}
