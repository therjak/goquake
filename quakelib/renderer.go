package quakelib

//typedef float vec3[3];
import "C"
import (
	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/model"
	"github.com/therjak/goquake/progs"
)

type fPlane struct {
	signBits uint8 // caching of plane side tests
	normal   vec.Vec3
	dist     float32
}

type qRenderer struct {
	frustum [4]fPlane
}

var renderer qRenderer

//export R_CullBox
func R_CullBox(mins, maxs *C.float) bool {
	return renderer.CullBox(p2v3(mins), p2v3(maxs))
}

// CullBox returns true if the box is completely outside the frustum
func (r *qRenderer) CullBox(mins, maxs vec.Vec3) bool {
	for _, f := range r.frustum {
		switch f.signBits {
		case 0:
			if f.normal[0]*maxs[0]+f.normal[1]*maxs[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 1:
			if f.normal[0]*mins[0]+f.normal[1]*maxs[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 2:
			if f.normal[0]*maxs[0]+f.normal[1]*mins[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 3:
			if f.normal[0]*mins[0]+f.normal[1]*mins[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 4:
			if f.normal[0]*maxs[0]+f.normal[1]*maxs[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		case 5:
			if f.normal[0]*mins[0]+f.normal[1]*maxs[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		case 6:
			if f.normal[0]*maxs[0]+f.normal[1]*mins[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		case 7:
			if f.normal[0]*mins[0]+f.normal[1]*mins[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		}
	}
	return false
}

func (p *fPlane) UpdateSignBits() {
	p.signBits = 0
	if p.normal[0] < 0 {
		p.signBits |= 1 << 0
	}
	if p.normal[1] < 0 {
		p.signBits |= 1 << 1
	}
	if p.normal[2] < 0 {
		p.signBits |= 1 << 2
	}
}

func deg2rad(a float32) float32 {
	a /= 180
	a *= math32.Pi
	return a
}

func (p *fPlane) TurnVector(forward, side vec.Vec3, angle float32) {
	ar := deg2rad(angle)
	scaleForward := math32.Cos(ar)
	scaleSide := math32.Sin(ar)
	p.normal = vec.Add(vec.Scale(scaleForward, forward), vec.Scale(scaleSide, side))
	p.dist = vec.Dot(qRefreshRect.viewOrg, p.normal)
	p.UpdateSignBits()
}

//export R_SetFrustum
func R_SetFrustum(fovx, fovy float32) {
	renderer.SetFrustum(fovx, fovy)
}

func (r *qRenderer) SetFrustum(fovx, fovy float32) {
	// We do not use qRefreshRect.fovX/fovY directly as water has an effect on these values
	r.frustum[0].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewRight, fovx/2-90) // left
	r.frustum[1].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewRight, 90-fovx/2) // right
	r.frustum[2].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewUp, 90-fovy/2)    // bottom
	r.frustum[3].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewUp, fovy/2-90)    // top
}

//export R_DrawViewModel
func R_DrawViewModel() {
	renderer.DrawWeaponModel()
}

func (r *qRenderer) DrawWeaponModel() {
	if !cvars.RDrawViewModel.Bool() ||
		!cvars.RDrawEntities.Bool() ||
		cvars.ChaseActive.Bool() {
		return
	}
	if cl.items&progs.ItemInvisibility != 0 ||
		cl.stats.health <= 0 {
		return
	}
	weapon := cl.WeaponEntity()
	if weapon.Model == nil {
		return
	}
	if weapon.Model.Type != model.ModAlias {
		// this fixes a crash
		return
	}

	// hack the depth range to prevent view model from poking into walls
	gl.DepthRange(0, 0.3)
	r.DrawAliasModel(weapon)
	gl.DepthRange(0, 1)
}
