// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import _ "embed"

//go:embed shader/position.vert
var vertexPositionSource string

//go:embed shader/world_position.vert
var vertexWorldPositionSource string

//go:embed shader/texture.vert
var vertexTextureSource string

//go:embed shader/texture2.vert
var vertexTextureSource2 string

//go:embed shader/dual_texture.vert
var vertexDualTextureSource string

//go:embed shader/dual_texture.frag
var fragmentSourceDualTextureDrawer string

//go:embed shader/texture.frag
var fragmentSourceDrawer string

//go:embed shader/color_rec.frag
var fragmentSourceColorRecDrawer string

//go:embed shader/post_process.frag
var postProcessFragment string

//go:embed shader/particle.vert
var vertexSourceParticleDrawer string

//go:embed shader/particle.frag
var fragmentSourceParticleDrawer string

//go:embed shader/cone.vert
var vertexConeSource string

//go:embed shader/cone.geom
var geometryConeSource string

//go:embed shader/cone.frag
var fragmentConeSource string

//go:embed shader/alias.vert
var vertexSourceAliasDrawer string

//go:embed shader/alias.frag
var fragmentSourceAliasDrawer string
