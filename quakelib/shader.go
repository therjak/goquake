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
	mat4 m = projection * modelview;
	vs_out.innerColor = innerColor;
	vs_out.outerColor = outerColor;
	vec4 c = m * vec4(position, 1.0);
	vec4 x = m * vec4(position + vec3(radius, 0, 0), 1);
	vec4 y = m * vec4(position + vec3(0, radius, 0), 1);
	vec4 z = m * vec4(position - vec3(0, 0, radius), 1);
	vs_out.radius.x = distance(c,x);
	vs_out.radius.y = distance(c,y);
	vs_out.radius.z = distance(c,z);
	gl_Position = c;
}
` + "\x00"

	geometryConeSource = `
#version 330 core
layout (points) in;
layout (triangle_strip, max_vertices = 17) out;

in VS_OUT {
  vec3 radius;
  vec3 innerColor;
  vec3 outerColor;
} gs_in[];

out vec3 Color;

void buildCone(vec4 position) {
  vec3 r = gs_in[0].radius;
	Color = gs_in[0].outerColor;
	gl_Position = position + vec4(-r.x, -r.y,0.0,0.0);
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = position + vec4(-r.x, r.y,0.0,0.0);
	EmitVertex();

	Color = gs_in[0].innerColor;
	gl_Position = position + vec4(r.x,-r.y,0.0,0.0);
	EmitVertex();

	Color = gs_in[0].outerColor;
	gl_Position = position + vec4(r.x,r.y,0.0,0.0);
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
