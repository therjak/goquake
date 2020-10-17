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

	vertexCircleSource = `
#version 330
layout (location = 0) in vec3 position;
layout (location = 1) in float radius;
layout (location = 2) in vec3 innerColor;
layout (location = 3) in vec3 outerColor;
out float Radius;
out vec3 InnerColor;
out vec3 OuterColor;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
  Radius = radius;
	InnerColor = innerColor;
	OuterColor = outerColor;
	gl_Position = projection * modelview * vec4(position, 1.0);
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

	fragmentCircleSource = `
#version 330
in float Radius;
in vec3 InnerColor;
in vec3 OuterColor;
out vec4 frag_color;

float circle(vec3 position, float radius) {
  // return 0 for radius > length(position), 1 otherwise
  return step(radius, length(position));
}

void main() {
  vec3 position = gl_FragCoord.xyz;
  vec3 color1 = vec3(0.2,0.1,0.0);
  vec3 color2 = vec3(0,0,0);
  float c = circle(position, 0.3);
  color1 = vec3(c);
  frag_color = vec4(color1, 1.0);
}
` + "\x00"
)
