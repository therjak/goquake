// THERJAK: this file should be ready to be converted
#include <stdio.h>

#include "quakedef.h"

#define ABSOLUTE_MIN_PARTICLES \
  512  // no fewer than this no matter what's
       //  on the command line

typedef enum {
  pt_static,
  pt_grav,
  pt_slowgrav,
  pt_fire,
  pt_explode,
  pt_explode2,
  pt_blob,
  pt_blob2
} ptype_t;

// !!! if this is changed, it must be changed in d_ifacea.h too !!!
typedef struct particle_s {
  // driver-usable fields
  vec3_t org;
  float color;
  // drivers never touch the following fields
  struct particle_s *next;
  vec3_t vel;
  float ramp;
  float die;
  ptype_t type;
} particle_t;

int ramp1[8] = {0x6f, 0x6d, 0x6b, 0x69, 0x67, 0x65, 0x63, 0x61};
int ramp2[8] = {0x6f, 0x6e, 0x6d, 0x6c, 0x6b, 0x6a, 0x68, 0x66};
int ramp3[8] = {0x6d, 0x6b, 6, 5, 4, 3};

particle_t *active_particles, *free_particles, *particles;

int r_numparticles;

uint32_t particletexture, particletexture1, particletexture2,
    particletexture3;      // johnfitz
float texturescalefactor;  // johnfitz -- compensate for apparent size of
                           // different particle textures

cvar_t r_particles;
cvar_t r_quadparticles;

/*
===============
R_ParticleTextureLookup -- johnfitz -- generate nice antialiased 32x32 circle
for particles
===============
*/
int R_ParticleTextureLookup(int x, int y, int sharpness) {
  int r;  // distance from point x,y to circle origin, squared
  int a;  // alpha value to return

  x -= 16;
  y -= 16;
  r = x * x + y * y;
  r = r > 255 ? 255 : r;
  a = sharpness * (255 - r);
  a = q_min(a, 255);
  return a;
}

/*
===============
R_InitParticleTextures -- johnfitz -- rewritten
===============
*/
void R_InitParticleTextures(void) {
  int x, y;
  static byte particle1_data[64 * 64 * 4];
  static byte particle2_data[2 * 2 * 4];
  static byte particle3_data[64 * 64 * 4];
  byte *dst;

  // particle texture 1 -- circle
  dst = particle1_data;
  for (x = 0; x < 64; x++)
    for (y = 0; y < 64; y++) {
      *dst++ = 255;
      *dst++ = 255;
      *dst++ = 255;
      *dst++ = R_ParticleTextureLookup(x, y, 8);
    }
  particletexture1 =
      TexMgrLoadParticleImage("particle1", 64, 64, particle1_data);

  // particle texture 2 -- square
  dst = particle2_data;
  for (x = 0; x < 2; x++)
    for (y = 0; y < 2; y++) {
      *dst++ = 255;
      *dst++ = 255;
      *dst++ = 255;
      *dst++ = x || y ? 0 : 255;
    }
  particletexture2 = TexMgrLoadParticleImage("particle2", 2, 2, particle2_data);

  // particle texture 3 -- blob
  dst = particle3_data;
  for (x = 0; x < 64; x++)
    for (y = 0; y < 64; y++) {
      *dst++ = 255;
      *dst++ = 255;
      *dst++ = 255;
      *dst++ = R_ParticleTextureLookup(x, y, 2);
    }
  particletexture3 =
      TexMgrLoadParticleImage("particle3", 64, 64, particle3_data);

  // set default
  particletexture = particletexture1;
  texturescalefactor = 1.27;
}

/*
===============
R_SetParticleTexture_f -- johnfitz
===============
*/
// THERJAK: cvar callback
static void R_SetParticleTexture_f(cvar_t *var) {
  switch ((int)(Cvar_GetValue(&r_particles))) {
    case 1:
      particletexture = particletexture1;
      texturescalefactor = 1.27;
      break;
    case 2:
      particletexture = particletexture2;
      texturescalefactor = 1.0;
      break;
      //	case 3:
      //		particletexture = particletexture3;
      //		texturescalefactor = 1.5;
      //		break;
  }
}

/*
===============
R_InitParticles
===============
*/
// THERJAK: external
void R_InitParticles(void) {
  r_numparticles = CMLParticles();
  if (r_numparticles < ABSOLUTE_MIN_PARTICLES) {
    r_numparticles = ABSOLUTE_MIN_PARTICLES;
  }

  particles = (particle_t *)Hunk_AllocName(r_numparticles * sizeof(particle_t),
                                           "particles");

  Cvar_FakeRegister(&r_particles, "r_particles");
  Cvar_SetCallback(&r_particles, R_SetParticleTexture_f);
  Cvar_FakeRegister(&r_quadparticles, "r_quadparticles");

  R_InitParticleTextures();  // johnfitz
}

/*
===============
R_EntityParticles
===============
*/
#define NUMVERTEXNORMALS 162
vec3_t avelocities[NUMVERTEXNORMALS];
float beamlength = 16;

// THERJAK: external
void R_EntityParticles(entity_t *ent) {
  int i;
  particle_t *p;
  float angle;
  float sp, sy, cp, cy;
  vec3_t forward;
  float dist;

  dist = 64;

  if (!avelocities[0][0]) {
    for (i = 0; i < NUMVERTEXNORMALS; i++) {
      avelocities[i][0] = (rand() & 255) * 0.01;
      avelocities[i][1] = (rand() & 255) * 0.01;
      avelocities[i][2] = (rand() & 255) * 0.01;
    }
  }

  for (i = 0; i < NUMVERTEXNORMALS; i++) {
    angle = CL_Time() * avelocities[i][0];
    sy = sin(angle);
    cy = cos(angle);
    angle = CL_Time() * avelocities[i][1];
    sp = sin(angle);
    cp = cos(angle);

    forward[0] = cp * cy;
    forward[1] = cp * sy;
    forward[2] = -sp;

    if (!free_particles) return;
    p = free_particles;
    free_particles = p->next;
    p->next = active_particles;
    active_particles = p;

    p->die = CL_Time() + 0.01;
    p->color = 0x6f;
    p->type = pt_explode;

    p->org[0] = ent->origin[0] + R_avertexnormals(i, 0) * dist +
                forward[0] * beamlength;
    p->org[1] = ent->origin[1] + R_avertexnormals(i, 1) * dist +
                forward[1] * beamlength;
    p->org[2] = ent->origin[2] + R_avertexnormals(i, 2) * dist +
                forward[2] * beamlength;
  }
}

/*
===============
R_ClearParticles
===============
*/
// THERJAK: external
void R_ClearParticles(void) {
  int i;

  free_particles = &particles[0];
  active_particles = NULL;

  for (i = 0; i < r_numparticles; i++) particles[i].next = &particles[i + 1];
  particles[r_numparticles - 1].next = NULL;
}

/*
===============
R_ReadPointFile_f
===============
*/
// THERJAK: cmd
void R_ReadPointFile_f(void) {
  // This is a file to debug maps. They should not be part of a pak.
  // It's to show ingame where the map has holes.
  FILE *f;
  vec3_t org;
  int r;
  int c;
  particle_t *p;
  char name[MAX_QPATH];

  if (CLS_GetState() != ca_connected) return;  // need an active map.

  q_snprintf(name, sizeof(name), "maps/%s.pts", cl.mapname);

  f = fopen(name, "r");
  if (!f) {
    Con_Printf("couldn't open %s\n", name);
    return;
  }

  Con_Printf("Reading %s...\n", name);
  c = 0;
  for (;;) {
    r = fscanf(f, "%f %f %f\n", &org[0], &org[1], &org[2]);
    if (r != 3) break;
    c++;

    if (!free_particles) {
      Con_Printf("Not enough free particles\n");
      break;
    }
    p = free_particles;
    free_particles = p->next;
    p->next = active_particles;
    active_particles = p;

    p->die = 99999;
    p->color = (-c) & 15;
    p->type = pt_static;
    VectorCopy(vec3_origin, p->vel);
    VectorCopy(org, p->org);
  }

  fclose(f);
  Con_Printf("%i points read\n", c);
}

/*
===============
R_ParticleExplosion
===============
*/
// THERJAK: external
void R_ParticleExplosion(vec3_t org) {
  int i, j;
  particle_t *p;

  for (i = 0; i < 1024; i++) {
    if (!free_particles) return;
    p = free_particles;
    free_particles = p->next;
    p->next = active_particles;
    active_particles = p;

    p->die = CL_Time() + 5;
    p->color = ramp1[0];
    p->ramp = rand() & 3;
    if (i & 1) {
      p->type = pt_explode;
      for (j = 0; j < 3; j++) {
        p->org[j] = org[j] + ((rand() % 32) - 16);
        p->vel[j] = (rand() % 512) - 256;
      }
    } else {
      p->type = pt_explode2;
      for (j = 0; j < 3; j++) {
        p->org[j] = org[j] + ((rand() % 32) - 16);
        p->vel[j] = (rand() % 512) - 256;
      }
    }
  }
}

/*
===============
R_ParticleExplosion2
===============
*/
// THERJAK: external
void R_ParticleExplosion2(vec3_t org, int colorStart, int colorLength) {
  int i, j;
  particle_t *p;
  int colorMod = 0;

  for (i = 0; i < 512; i++) {
    if (!free_particles) return;
    p = free_particles;
    free_particles = p->next;
    p->next = active_particles;
    active_particles = p;

    p->die = CL_Time() + 0.3;
    p->color = colorStart + (colorMod % colorLength);
    colorMod++;

    p->type = pt_blob;
    for (j = 0; j < 3; j++) {
      p->org[j] = org[j] + ((rand() % 32) - 16);
      p->vel[j] = (rand() % 512) - 256;
    }
  }
}

/*
===============
R_BlobExplosion
===============
*/
// THERJAK: external
void R_BlobExplosion(vec3_t org) {
  int i, j;
  particle_t *p;

  for (i = 0; i < 1024; i++) {
    if (!free_particles) return;
    p = free_particles;
    free_particles = p->next;
    p->next = active_particles;
    active_particles = p;

    p->die = CL_Time() + 1 + (rand() & 8) * 0.05;

    if (i & 1) {
      p->type = pt_blob;
      p->color = 66 + rand() % 6;
      for (j = 0; j < 3; j++) {
        p->org[j] = org[j] + ((rand() % 32) - 16);
        p->vel[j] = (rand() % 512) - 256;
      }
    } else {
      p->type = pt_blob2;
      p->color = 150 + rand() % 6;
      for (j = 0; j < 3; j++) {
        p->org[j] = org[j] + ((rand() % 32) - 16);
        p->vel[j] = (rand() % 512) - 256;
      }
    }
  }
}

/*
===============
R_RunParticleEffect
===============
*/
// THERJAK: external
void R_RunParticleEffect(vec3_t org, vec3_t dir, int color, int count) {
  int i, j;
  particle_t *p;

  for (i = 0; i < count; i++) {
    if (!free_particles) return;
    p = free_particles;
    free_particles = p->next;
    p->next = active_particles;
    active_particles = p;

    if (count == 1024) {  // rocket explosion
      p->die = CL_Time() + 5;
      p->color = ramp1[0];
      p->ramp = rand() & 3;
      if (i & 1) {
        p->type = pt_explode;
        for (j = 0; j < 3; j++) {
          p->org[j] = org[j] + ((rand() % 32) - 16);
          p->vel[j] = (rand() % 512) - 256;
        }
      } else {
        p->type = pt_explode2;
        for (j = 0; j < 3; j++) {
          p->org[j] = org[j] + ((rand() % 32) - 16);
          p->vel[j] = (rand() % 512) - 256;
        }
      }
    } else {
      p->die = CL_Time() + 0.1 * (rand() % 5);
      p->color = (color & ~7) + (rand() & 7);
      p->type = pt_slowgrav;
      for (j = 0; j < 3; j++) {
        p->org[j] = org[j] + ((rand() & 15) - 8);
        p->vel[j] = dir[j] * 15;  // + (rand()%300)-150;
      }
    }
  }
}

/*
===============
R_LavaSplash
===============
*/
// THERJAK: external
void R_LavaSplash(vec3_t org) {
  int i, j, k;
  particle_t *p;
  float vel;
  vec3_t dir;

  for (i = -16; i < 16; i++)
    for (j = -16; j < 16; j++)
      for (k = 0; k < 1; k++) {
        if (!free_particles) return;
        p = free_particles;
        free_particles = p->next;
        p->next = active_particles;
        active_particles = p;

        p->die = CL_Time() + 2 + (rand() & 31) * 0.02;
        p->color = 224 + (rand() & 7);
        p->type = pt_slowgrav;

        dir[0] = j * 8 + (rand() & 7);
        dir[1] = i * 8 + (rand() & 7);
        dir[2] = 256;

        p->org[0] = org[0] + dir[0];
        p->org[1] = org[1] + dir[1];
        p->org[2] = org[2] + (rand() & 63);

        VectorNormalize(dir);
        vel = 50 + (rand() & 63);
        VectorScale(dir, vel, p->vel);
      }
}

/*
===============
R_TeleportSplash
===============
*/
// THERJAK: external
void R_TeleportSplash(vec3_t org) {
  int i, j, k;
  particle_t *p;
  float vel;
  vec3_t dir;

  for (i = -16; i < 16; i += 4)
    for (j = -16; j < 16; j += 4)
      for (k = -24; k < 32; k += 4) {
        if (!free_particles) return;
        p = free_particles;
        free_particles = p->next;
        p->next = active_particles;
        active_particles = p;

        p->die = CL_Time() + 0.2 + (rand() & 7) * 0.02;
        p->color = 7 + (rand() & 7);
        p->type = pt_slowgrav;

        dir[0] = j * 8;
        dir[1] = i * 8;
        dir[2] = k * 8;

        p->org[0] = org[0] + i + (rand() & 3);
        p->org[1] = org[1] + j + (rand() & 3);
        p->org[2] = org[2] + k + (rand() & 3);

        VectorNormalize(dir);
        vel = 50 + (rand() & 63);
        VectorScale(dir, vel, p->vel);
      }
}

/*
===============
R_RocketTrail

FIXME -- rename function and use #defined types instead of numbers
===============
*/
// THERJAK: external
void R_RocketTrail(vec3_t start, vec3_t end, int type) {
  vec3_t vec;
  float len;
  int j;
  particle_t *p;
  int dec;
  static int tracercount;

  VectorSubtract(end, start, vec);
  len = VectorNormalize(vec);
  if (type < 128)
    dec = 3;
  else {
    dec = 1;
    type -= 128;
  }

  while (len > 0) {
    len -= dec;

    if (!free_particles) return;
    p = free_particles;
    free_particles = p->next;
    p->next = active_particles;
    active_particles = p;

    VectorCopy(vec3_origin, p->vel);
    p->die = CL_Time() + 2;

    switch (type) {
      case 0:  // rocket trail
        p->ramp = (rand() & 3);
        p->color = ramp3[(int)p->ramp];
        p->type = pt_fire;
        for (j = 0; j < 3; j++) p->org[j] = start[j] + ((rand() % 6) - 3);
        break;

      case 1:  // smoke smoke
        p->ramp = (rand() & 3) + 2;
        p->color = ramp3[(int)p->ramp];
        p->type = pt_fire;
        for (j = 0; j < 3; j++) p->org[j] = start[j] + ((rand() % 6) - 3);
        break;

      case 2:  // blood
        p->type = pt_grav;
        p->color = 67 + (rand() & 3);
        for (j = 0; j < 3; j++) p->org[j] = start[j] + ((rand() % 6) - 3);
        break;

      case 3:
      case 5:  // tracer
        p->die = CL_Time() + 0.5;
        p->type = pt_static;
        if (type == 3)
          p->color = 52 + ((tracercount & 4) << 1);
        else
          p->color = 230 + ((tracercount & 4) << 1);

        tracercount++;

        VectorCopy(start, p->org);
        if (tracercount & 1) {
          p->vel[0] = 30 * vec[1];
          p->vel[1] = 30 * -vec[0];
        } else {
          p->vel[0] = 30 * -vec[1];
          p->vel[1] = 30 * vec[0];
        }
        break;

      case 4:  // slight blood
        p->type = pt_grav;
        p->color = 67 + (rand() & 3);
        for (j = 0; j < 3; j++) p->org[j] = start[j] + ((rand() % 6) - 3);
        len -= 3;
        break;

      case 6:  // voor trail
        p->color = 9 * 16 + 8 + (rand() & 3);
        p->type = pt_static;
        p->die = CL_Time() + 0.3;
        for (j = 0; j < 3; j++) p->org[j] = start[j] + ((rand() & 15) - 8);
        break;
    }

    VectorAdd(start, vec, start);
  }
}

/*
===============
CL_RunParticles -- johnfitz -- all the particle behavior, separated from
R_DrawParticles
===============
*/
// THERJAK: external
void CL_RunParticles(void) {
  particle_t *p, *kill;
  int i;
  float time1, time2, time3, dvel, frametime, grav;
  extern cvar_t sv_gravity;

  frametime = CL_Time() - CL_OldTime();
  time3 = frametime * 15;
  time2 = frametime * 10;
  time1 = frametime * 5;
  grav = frametime * Cvar_GetValue(&sv_gravity) * 0.05;
  dvel = 4 * frametime;

  for (;;) {
    kill = active_particles;
    if (kill && kill->die < CL_Time()) {
      active_particles = kill->next;
      kill->next = free_particles;
      free_particles = kill;
      continue;
    }
    break;
  }

  for (p = active_particles; p; p = p->next) {
    for (;;) {
      kill = p->next;
      if (kill && kill->die < CL_Time()) {
        p->next = kill->next;
        kill->next = free_particles;
        free_particles = kill;
        continue;
      }
      break;
    }

    p->org[0] += p->vel[0] * frametime;
    p->org[1] += p->vel[1] * frametime;
    p->org[2] += p->vel[2] * frametime;

    switch (p->type) {
      case pt_static:
        break;
      case pt_fire:
        p->ramp += time1;
        if (p->ramp >= 6)
          p->die = -1;
        else
          p->color = ramp3[(int)p->ramp];
        p->vel[2] += grav;
        break;

      case pt_explode:
        p->ramp += time2;
        if (p->ramp >= 8)
          p->die = -1;
        else
          p->color = ramp1[(int)p->ramp];
        for (i = 0; i < 3; i++) p->vel[i] += p->vel[i] * dvel;
        p->vel[2] -= grav;
        break;

      case pt_explode2:
        p->ramp += time3;
        if (p->ramp >= 8)
          p->die = -1;
        else
          p->color = ramp2[(int)p->ramp];
        for (i = 0; i < 3; i++) p->vel[i] -= p->vel[i] * frametime;
        p->vel[2] -= grav;
        break;

      case pt_blob:
        for (i = 0; i < 3; i++) p->vel[i] += p->vel[i] * dvel;
        p->vel[2] -= grav;
        break;

      case pt_blob2:
        for (i = 0; i < 2; i++) p->vel[i] -= p->vel[i] * dvel;
        p->vel[2] -= grav;
        break;

      case pt_grav:
      case pt_slowgrav:
        p->vel[2] -= grav;
        break;
    }
  }
}

/*
===============
R_DrawParticles -- johnfitz -- moved all non-drawing code to CL_RunParticles
===============
*/
// THERJAK: external
void R_DrawParticles(void) {
  particle_t *p;
  float scale;
  vec3_t up, right, p_up, p_right, p_upright;  // johnfitz -- p_ vectors
  GLubyte color[4], *c;       // johnfitz -- particle transparency
  extern cvar_t r_particles;  // johnfitz
  // float			alpha; //johnfitz -- particle transparency

  if (!Cvar_GetValue(&r_particles)) return;

  // ericw -- avoid empty glBegin(),glEnd() pair below; causes issues on AMD
  if (!active_particles) return;

  VectorScale(vup, 1.5, up);
  VectorScale(vright, 1.5, right);

  GLBind(particletexture);
  glEnable(GL_BLEND);
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_MODULATE);
  glDepthMask(GL_FALSE);  // johnfitz -- fix for particle z-buffer bug

  if (Cvar_GetValue(&r_quadparticles))  // johnitz -- quads save fillrate
  {
    glBegin(GL_QUADS);
    for (p = active_particles; p; p = p->next) {
      // hack a scale up to keep particles from disapearing
      scale = (p->org[0] - r_origin[0]) * vpn[0] +
              (p->org[1] - r_origin[1]) * vpn[1] +
              (p->org[2] - r_origin[2]) * vpn[2];
      if (scale < 20)
        scale = 1 + 0.08;  // johnfitz -- added .08 to be consistent
      else
        scale = 1 + scale * 0.004;

      scale /= 2.0;  // quad is half the size of triangle

      scale *= texturescalefactor;  // johnfitz -- compensate for apparent size
                                    // of different particle textures

      // johnfitz -- particle transparency and fade out
      c = (GLubyte *)&d_8to24table[(int)p->color];
      color[0] = c[0];
      color[1] = c[1];
      color[2] = c[2];
      // alpha = CLAMP(0, p->die + 0.5 - CL_Time(), 1);
      color[3] = 255;  //(int)(alpha * 255);
      glColor4ubv(color);
      // johnfitz

      glTexCoord2f(0, 0);
      glVertex3fv(p->org);

      glTexCoord2f(0.5, 0);
      VectorMA(p->org, scale, up, p_up);
      glVertex3fv(p_up);

      glTexCoord2f(0.5, 0.5);
      VectorMA(p_up, scale, right, p_upright);
      glVertex3fv(p_upright);

      glTexCoord2f(0, 0.5);
      VectorMA(p->org, scale, right, p_right);
      glVertex3fv(p_right);

      rs_particles++;  // johnfitz //FIXME: just use r_numparticles
    }
    glEnd();
  } else  // johnitz --  triangles save verts
  {
    glBegin(GL_TRIANGLES);
    for (p = active_particles; p; p = p->next) {
      // hack a scale up to keep particles from disapearing
      scale = (p->org[0] - r_origin[0]) * vpn[0] +
              (p->org[1] - r_origin[1]) * vpn[1] +
              (p->org[2] - r_origin[2]) * vpn[2];
      if (scale < 20)
        scale = 1 + 0.08;  // johnfitz -- added .08 to be consistent
      else
        scale = 1 + scale * 0.004;

      scale *= texturescalefactor;  // johnfitz -- compensate for apparent size
                                    // of different particle textures

      // johnfitz -- particle transparency and fade out
      c = (GLubyte *)&d_8to24table[(int)p->color];
      color[0] = c[0];
      color[1] = c[1];
      color[2] = c[2];
      // alpha = CLAMP(0, p->die + 0.5 - CL_Time(), 1);
      color[3] = 255;  //(int)(alpha * 255);
      glColor4ubv(color);
      // johnfitz

      glTexCoord2f(0, 0);
      glVertex3fv(p->org);

      glTexCoord2f(1, 0);
      VectorMA(p->org, scale, up, p_up);
      glVertex3fv(p_up);

      glTexCoord2f(0, 1);
      VectorMA(p->org, scale, right, p_right);
      glVertex3fv(p_right);

      rs_particles++;  // johnfitz //FIXME: just use r_numparticles
    }
    glEnd();
  }

  glDepthMask(GL_TRUE);  // johnfitz -- fix for particle z-buffer bug
  glDisable(GL_BLEND);
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE);
  glColor3f(1, 1, 1);
}

/*
===============
R_DrawParticles_ShowTris -- johnfitz
===============
*/
// THERJAK: external
void R_DrawParticles_ShowTris(void) {
  particle_t *p;
  float scale;
  vec3_t up, right, p_up, p_right, p_upright;
  extern cvar_t r_particles;

  if (!Cvar_GetValue(&r_particles)) return;

  VectorScale(vup, 1.5, up);
  VectorScale(vright, 1.5, right);

  if (Cvar_GetValue(&r_quadparticles)) {
    for (p = active_particles; p; p = p->next) {
      glBegin(GL_TRIANGLE_FAN);

      // hack a scale up to keep particles from disapearing
      scale = (p->org[0] - r_origin[0]) * vpn[0] +
              (p->org[1] - r_origin[1]) * vpn[1] +
              (p->org[2] - r_origin[2]) * vpn[2];
      if (scale < 20)
        scale = 1 + 0.08;  // johnfitz -- added .08 to be consistent
      else
        scale = 1 + scale * 0.004;

      scale /= 2.0;  // quad is half the size of triangle

      scale *= texturescalefactor;  // compensate for apparent size of different
                                    // particle textures

      glVertex3fv(p->org);

      VectorMA(p->org, scale, up, p_up);
      glVertex3fv(p_up);

      VectorMA(p_up, scale, right, p_upright);
      glVertex3fv(p_upright);

      VectorMA(p->org, scale, right, p_right);
      glVertex3fv(p_right);

      glEnd();
    }
  } else {
    glBegin(GL_TRIANGLES);
    for (p = active_particles; p; p = p->next) {
      // hack a scale up to keep particles from disapearing
      scale = (p->org[0] - r_origin[0]) * vpn[0] +
              (p->org[1] - r_origin[1]) * vpn[1] +
              (p->org[2] - r_origin[2]) * vpn[2];
      if (scale < 20)
        scale = 1 + 0.08;  // johnfitz -- added .08 to be consistent
      else
        scale = 1 + scale * 0.004;

      scale *= texturescalefactor;  // compensate for apparent size of different
                                    // particle textures

      glVertex3fv(p->org);

      VectorMA(p->org, scale, up, p_up);
      glVertex3fv(p_up);

      VectorMA(p->org, scale, right, p_right);
      glVertex3fv(p_right);
    }
    glEnd();
  }
}
