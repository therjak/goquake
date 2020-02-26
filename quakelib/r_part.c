#include <stdio.h>

#include "quakedef.h"
typedef enum {
  pt_static,
} ptype_t;

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

particle_t *active_particles, *free_particles, *particles;

/*
cvar_t r_particles;
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
  }
}

void R_InitParticles(void) {
  Cvar_FakeRegister(&r_particles, "r_particles");
  Cvar_SetCallback(&r_particles, R_SetParticleTexture_f);
}*/

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
