// SPDX-License-Identifier: GPL-2.0-or-later

// package input handles button event tracking
package input

import (
	"goquake/cbuf"
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

func (b *button) upCmd() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		k := a.Args()[1:]
		if len(k) == 0 {
			// typed manually
			b.holdingDown[0] = 0
			b.holdingDown[1] = 0
			b.down = false
			b.impulseDown = false
			b.impulseUp = true
		} else {
			b.upKey(k[0].Int())
		}
		return nil
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

func (b *button) downCmd() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		k := a.Args()[1:]
		if len(k) == 0 {
			// typed manually
			b.downKey(-1)
		} else {
			b.downKey(k[0].Int())
		}
		return nil
	}
}

func Commands(c *cmd.Commands) error {
	// Key events issue these commands and pass the key number as argument,
	// if no number expect console/cfg input
	if err := c.Add("+moveup", Up.downCmd()); err != nil {
		return err
	}
	if err := c.Add("-moveup", Up.upCmd()); err != nil {
		return err
	}
	if err := c.Add("+movedown", Down.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-movedown", Down.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+left", Left.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-left", Left.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+right", Right.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-right", Right.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+forward", Forward.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-forward", Forward.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+back", Back.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-back", Back.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+lookup", LookUp.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-lookup", LookUp.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+lookdown", LookDown.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-lookdown", LookDown.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+strafe", Strafe.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-strafe", Strafe.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+moveleft", MoveLeft.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-moveleft", MoveLeft.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+moveright", MoveRight.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-moveright", MoveRight.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+speed", Speed.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-speed", Speed.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+attack", Attack.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-attack", Attack.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+use", Use.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-use", Use.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+jump", Jump.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-jump", Jump.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+klook", KLook.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-klook", KLook.upCmd()); err != nil {
		return nil
	}
	if err := c.Add("+mlook", MLook.downCmd()); err != nil {
		return nil
	}
	if err := c.Add("-mlook", MLook.upCmd()); err != nil {
		// TODO: this command did contain the following as well:
		// if !MLook.down && Cvar_GetValue(&lookspring) {
		// V_StartPitchDrift()
		// }
		return nil
	}
	return nil
}
