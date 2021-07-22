// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

//void R_UpdateWarpTexturesC(void);
import "C"

import (
	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
	"goquake/cvars"
	"goquake/math"
)

//export R_UpdateWarpTextures
func R_UpdateWarpTextures() {
	updateWarpTextures()
}

func warpcalc(s, t float32) float32 {
	return (s + turbsin[int((t*2)+(float32(cl.time)*(128.0/math32.Pi)))&255]) * 1.0 / 64.0
}

func updateWarpTextures() {
	if cvars.ROldWater.Bool() || cl.paused {
		return
	}
	warptess := 128.0 / math.Clamp32(3.0, cvars.RWaterQuality.Value(), 64.0)
	for _, tx := range cl.worldModel.Textures {
		if tx == nil {
			continue
		}
		if !tx.UpdateWarp {
			continue
		}
		qCanvas.Set(CANVAS_WARPIMAGE)
		tx.Texture.Bind()

		var x2 float32 // the end of the tile
		for x := float32(0.0); x < 128.0; x = x2 {
			x2 = x + warptess
			// glbegin triangle_strip
			for y := float32(0.0); y < 128.01; y += warptess {
				// TODO(therjak): move the txcoord calc into the shader
				// txcoord(warpcalc(x,y), warpcalc(y,x))
				// vertex(x,y)
				// txcoord(warpcalc(x2,y), warpcalc(y,x2))
				// vertex(x2,y)
			}
			// glEnd
		}

		tx.Warp.Bind()
		gl.CopyTexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, 0,
			int32(screen.Height)-glWarpImageSize, glWarpImageSize, glWarpImageSize)

		tx.UpdateWarp = false
	}

	qCanvas.Set(CANVAS_DEFAULT)

	if int(glWarpImageSize)+statusbar.Lines() > screen.Height {
		// The warp image also changed the statusbar area. Fix it.
		statusbar.MarkChanged()
	}

	screen.ResetTileClearUpdates()

	C.R_UpdateWarpTexturesC()
}
