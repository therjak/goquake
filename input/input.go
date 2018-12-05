// package input handles button event tracking
package input

import (
	"quake/cmd"
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
	cmd.AddCommand("+moveup", func(a []cmd.QArg) { Up.downCmd(a) })
	cmd.AddCommand("-moveup", func(a []cmd.QArg) { Up.upCmd(a) })
	cmd.AddCommand("+movedown", func(a []cmd.QArg) { Down.downCmd(a) })
	cmd.AddCommand("-movedown", func(a []cmd.QArg) { Down.upCmd(a) })
	cmd.AddCommand("+left", func(a []cmd.QArg) { Left.downCmd(a) })
	cmd.AddCommand("-left", func(a []cmd.QArg) { Left.upCmd(a) })
	cmd.AddCommand("+right", func(a []cmd.QArg) { Right.downCmd(a) })
	cmd.AddCommand("-right", func(a []cmd.QArg) { Right.upCmd(a) })
	cmd.AddCommand("+forward", func(a []cmd.QArg) { Forward.downCmd(a) })
	cmd.AddCommand("-forward", func(a []cmd.QArg) { Forward.upCmd(a) })
	cmd.AddCommand("+back", func(a []cmd.QArg) { Back.downCmd(a) })
	cmd.AddCommand("-back", func(a []cmd.QArg) { Back.upCmd(a) })
	cmd.AddCommand("+lookup", func(a []cmd.QArg) { LookUp.downCmd(a) })
	cmd.AddCommand("-lookup", func(a []cmd.QArg) { LookUp.upCmd(a) })
	cmd.AddCommand("+lookdown", func(a []cmd.QArg) { LookDown.downCmd(a) })
	cmd.AddCommand("-lookdown", func(a []cmd.QArg) { LookDown.upCmd(a) })
	cmd.AddCommand("+strafe", func(a []cmd.QArg) { Strafe.downCmd(a) })
	cmd.AddCommand("-strafe", func(a []cmd.QArg) { Strafe.upCmd(a) })
	cmd.AddCommand("+moveleft", func(a []cmd.QArg) { MoveLeft.downCmd(a) })
	cmd.AddCommand("-moveleft", func(a []cmd.QArg) { MoveLeft.upCmd(a) })
	cmd.AddCommand("+moveright", func(a []cmd.QArg) { MoveRight.downCmd(a) })
	cmd.AddCommand("-moveright", func(a []cmd.QArg) { MoveRight.upCmd(a) })
	cmd.AddCommand("+speed", func(a []cmd.QArg) { Speed.downCmd(a) })
	cmd.AddCommand("-speed", func(a []cmd.QArg) { Speed.upCmd(a) })
	cmd.AddCommand("+attack", func(a []cmd.QArg) { Attack.downCmd(a) })
	cmd.AddCommand("-attack", func(a []cmd.QArg) { Attack.upCmd(a) })
	cmd.AddCommand("+use", func(a []cmd.QArg) { Use.downCmd(a) })
	cmd.AddCommand("-use", func(a []cmd.QArg) { Use.upCmd(a) })
	cmd.AddCommand("+jump", func(a []cmd.QArg) { Jump.downCmd(a) })
	cmd.AddCommand("-jump", func(a []cmd.QArg) { Jump.upCmd(a) })
	cmd.AddCommand("+klook", func(a []cmd.QArg) { KLook.downCmd(a) })
	cmd.AddCommand("-klook", func(a []cmd.QArg) { KLook.upCmd(a) })
	cmd.AddCommand("+mlook", func(a []cmd.QArg) { MLook.downCmd(a) })
	cmd.AddCommand("-mlook", func(a []cmd.QArg) {
		MLook.upCmd(a)
		// if !MLook.down && Cvar_GetValue(&lookspring) {
		// V_StartPitchDrift()
		// }
	})
}
