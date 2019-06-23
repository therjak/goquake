// sv_main.c -- server main program

#include "_cgo_export.h"
//
#include "quakedef.h"

server_t sv;

extern qboolean pr_alpha_supported;  // johnfitz

//============================================================================

cvar_t sv_gravity;
/*
===============
SV_Init
===============
*/
void SV_Init(void) {
  int i;
  const char *p;

  Cvar_FakeRegister(&sv_gravity, "sv_gravity");

  SV_Init_Go();
}

/*
=============================================================================

The PVS must include a small area around the client to allow head bobbing
or other small motion on the client side.  Otherwise, a bob might cause an
entity that should be visible to not show up, especially when the bob
crosses a waterline.

=============================================================================
*/

int fatbytes;
byte fatpvs[MAX_MAP_LEAFS / 8];

void SV_AddToFatPVS(vec3_t org, mnode_t *node, qmodel_t *worldmodel)
{
  int i;
  byte *pvs;
  mplane_t *plane;
  float d;

  while (1) {
    // if this is a leaf, accumulate the pvs bits
    if (node->contents < 0) {
      if (node->contents != CONTENTS_SOLID) {
        pvs = Mod_LeafPVS((mleaf_t *)node, worldmodel);  
        for (i = 0; i < fatbytes; i++) fatpvs[i] |= pvs[i];
      }
      return;
    }

    plane = node->plane;
    d = DotProduct(org, plane->normal) - plane->dist;
    if (d > 8)
      node = node->children[0];
    else if (d < -8)
      node = node->children[1];
    else {  // go down both
      SV_AddToFatPVS(org, node->children[0], worldmodel);
      node = node->children[1];
    }
  }
}
/*
=============
SV_FatPVS

Calculates a PVS that is the inclusive or of all leafs within 8 pixels of the
given point.
=============
*/
byte *SV_FatPVS(
    vec3_t org,
    qmodel_t *worldmodel)
{
  fatbytes = (worldmodel->numleafs + 31) >> 3;
  Q_memset(fatpvs, 0, fatbytes);
  SV_AddToFatPVS(org, worldmodel->nodes, worldmodel);
  return fatpvs;
}

const char *SV_Name() {
  static char buffer[2048];
  char *s = SV_NameInt();
  strncpy(buffer, s, 2048);
  free(s);
  return buffer;
}

const char *SV_ModelName() {
  static char buffer[2048];
  char *s = SV_ModelNameInt();
  strncpy(buffer, s, 2048);
  free(s);
  return buffer;
}
