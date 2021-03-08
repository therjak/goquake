// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

const (
	vertexPositionSource = `
#version 330
layout (location = 0) in vec3 position;

void main() {
	gl_Position = vec4(position, 1.0);
}
` + "\x00"

	vertexTextureSource = `
#version 330
layout (location = 0) in vec3 position;
layout (location = 1) in vec2 texcoord;
out vec2 Texcoord;

void main() {
	Texcoord = texcoord;
	gl_Position = vec4(position, 1.0);
}
` + "\x00"

	vertexDualTextureSource = `
#version 330
layout (location = 0) in vec3 position;
layout (location = 1) in vec2 solidtexcoord;
layout (location = 2) in vec2 alphatexcoord;
out vec2 SolidTexcoord;
out vec2 AlphaTexcoord;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
	SolidTexcoord = solidtexcoord;
	AlphaTexcoord = alphatexcoord;
	gl_Position = projection * modelview * vec4(position, 1.0);
}
` + "\x00"

	vertexSourceParticleDrawer = `
#version 330
layout (location = 0) in vec3 vposition;
layout (location = 1) in vec2 vtexcoord;
layout (location = 2) in vec3 vcolor;
out vec2 Texcoord;
out vec3 InColor;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
	Texcoord = vtexcoord;
	InColor = vcolor;
	gl_Position = projection * modelview * vec4(vposition, 1.0);
}
` + "\x00"

	vertexConeSource = `
#version 330
layout (location = 0) in vec3 position;
layout (location = 1) in float radius;
layout (location = 2) in vec3 innerColor;
layout (location = 3) in vec3 outerColor;
out VS_OUT {
  vec3 radius;
  vec3 innerColor;
  vec3 outerColor;
} vs_out;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
	vs_out.innerColor = innerColor;
	vs_out.outerColor = outerColor;
	vec4 mc = modelview * vec4(position, 1.0);
	float x = projection[0][0] * (mc.x + radius);
	float y = projection[1][1] * (mc.y + radius);
	gl_Position = projection * mc;
	vs_out.radius.x = abs(gl_Position.x-x);
	vs_out.radius.y = abs(gl_Position.y-y);
	// No need to consider the frustum for the z radius.
	vs_out.radius.z = radius/gl_Position.w;
}
` + "\x00"

	geometryConeSource = `
#version 330 core
layout (points) in;
layout (triangle_strip, max_vertices = 32) out; // 8*4 vertices

const float M_S45 = 0.70710678118;
const float M_S225 = 0.38268343236;
const float M_S675 = 0.92387953251;

in VS_OUT {
  vec3 radius;
  vec3 innerColor;
  vec3 outerColor;
} gs_in[];

out vec3 Color;

void buildCone(vec4 position) {
  vec3 r = gs_in[0].radius;
	vec4 front = position - vec4(0.0, 0.0, r.z, 0.0);
	vec2 p1 = vec2(r.x * M_S675, r.y * M_S225);
	vec2 p2 = vec2(r.x * M_S45, r.y * M_S45);
	vec2 p3 = vec2(r.x * M_S225, r.y * M_S675);

	Color = gs_in[0].outerColor;
	// Part 1
	gl_Position = position + vec4(r.x, 0.0, 0.0, 0.0);
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(p1.x, p1.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(p2.x, p2.y, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

  // Part 2
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(p3.x, p3.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(0.0, r.y, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

  // Part 3
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(-p3.x, p3.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(-p2.x, p2.y, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

	// Part 4
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(-p1.x, p1.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(-r.x, 0, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

	// Part 5
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(-p1.x, -p1.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(-p2.x, -p2.y, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

  // Part 6
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(-p3.x, -p3.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(0.0, -r.y, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

  // Part 7
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(p3.x, -p3.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(p2.x, -p2.y, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

	// Part 8
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = front;
	EmitVertex();
	Color = gs_in[0].outerColor;

	gl_Position = position + vec4(p1.x, -p1.y, 0.0, 0.0);
	EmitVertex();

	gl_Position = position + vec4(r.x, 0, 0.0, 0.0);
	EmitVertex();

  EndPrimitive();

}

void main() {
  buildCone(gl_in[0].gl_Position);
}
` + "\x00"

	fragmentSourceDrawer = `
#version 330
in vec2 Texcoord;
out vec4 frag_color;
uniform sampler2D tex;

void main() {
  vec4 color = texture(tex, Texcoord);
	if (color.a < 0.666)
	  discard;
  frag_color = color;
}
` + "\x00"

	fragmentSourceDualTextureDrawer = `
#version 330
in vec2 SolidTexcoord;
in vec2 AlphaTexcoord;
out vec4 frag_color;
uniform sampler2D solid;
uniform sampler2D alpha;

void main() {
  vec4 color1 = texture(solid, SolidTexcoord);
	vec4 color2 = texture(alpha, AlphaTexcoord);
	// TODO: add fog
	// Blend vec4(Fog_GetColor, skyfog)
	frag_color = mix(color1, color2, color2.a);
}
` + "\x00"

	fragmentSourceColorRecDrawer = `
#version 330
in vec2 Texcoord;
out vec4 frag_color;
uniform vec4 in_color;

void main() {
  frag_color = in_color;
}
` + "\x00"

	postProcessFragment = `
#version 330
in vec2 Texcoord;
out vec4 frag_color;
uniform sampler2D tex;
uniform float gamma;
uniform float contrast;

void main() {
  vec4 color = texture(tex, Texcoord);
	color.rgb = color.rgb * contrast;
  frag_color = vec4(pow(color.rgb, vec3(gamma)), 1.0);
}
` + "\x00"

	fragmentSourceParticleDrawer = `
#version 330
in vec2 Texcoord;
in vec3 InColor;
out vec4 frag_color;
uniform sampler2D tex;

void main() {
	float color = texture(tex, Texcoord).r;
	frag_color.rgb = InColor;
	frag_color.a = color; // texture has only one chan
	frag_color = clamp(frag_color, vec4(0,0,0,0), vec4(1,1,1,1));
}
` + "\x00"

	fragmentConeSource = `
#version 330
in vec3 Color;
out vec4 frag_color;

void main() {
  frag_color = vec4(Color, 1.0);
}
` + "\x00"
)
