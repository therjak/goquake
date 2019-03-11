// world.c -- world query functions

#include "quakedef.h"

// FIXME: remove this mess!
#define EDICT_FROM_AREA(l) \
  ((edict_t *)((byte *)l - (intptr_t) & (((edict_t *)0)->area2)))

/*

entities never clip against themselves, or their owner

line of sight checks trace->crosscontent, but bullets don't

*/

typedef struct {
  vec3_t boxmins, boxmaxs;  // enclose the test object along entire move
  float *mins, *maxs;       // size of the moving object
  vec3_t mins2, maxs2;      // size when clipping against mosnters
  float *start, *end;
  trace_t trace;
  int type;
  int passedict;
} moveclip_t;

int SV_HullPointContents(hull_t *hull, int num, vec3_t p);

/*
===============================================================================

HULL BOXES

===============================================================================
*/

static hull_t box_hull;
static mclipnode_t box_clipnodes[6];  // johnfitz -- was dclipnode_t
static mplane_t box_planes[6];

/*
===================
SV_InitBoxHull

Set up the planes and clipnodes so that the six floats of a bounding box
can just be stored out and get a proper hull_t structure.
===================
*/
void SV_InitBoxHull(void) {
  int i;
  int side;

  box_hull.clipnodes = box_clipnodes;
  box_hull.planes = box_planes;
  box_hull.firstclipnode = 0;
  box_hull.lastclipnode = 5;

  for (i = 0; i < 6; i++) {
    box_clipnodes[i].planenum = i;

    side = i & 1;

    box_clipnodes[i].children[side] = CONTENTS_EMPTY;
    if (i != 5)
      box_clipnodes[i].children[side ^ 1] = i + 1;
    else
      box_clipnodes[i].children[side ^ 1] = CONTENTS_SOLID;

    box_planes[i].type = i >> 1;
    box_planes[i].normal[i >> 1] = 1;
  }
}

/*
===================
SV_HullForBox

To keep everything totally uniform, bounding boxes are turned into small
BSP trees instead of being compared directly.
===================
*/
hull_t *SV_HullForBox(vec3_t mins, vec3_t maxs) {
  box_planes[0].dist = maxs[0];
  box_planes[1].dist = mins[0];
  box_planes[2].dist = maxs[1];
  box_planes[3].dist = mins[1];
  box_planes[4].dist = maxs[2];
  box_planes[5].dist = mins[2];

  return &box_hull;
}

/*
================
SV_HullForEntity

Returns a hull that can be used for testing or clipping an object of mins/maxs
size.
Offset is filled in to contain the adjustment that must be added to the
testing object's origin to get a point to use with the returned hull.
================
*/
hull_t *SV_HullForEntity(entvars_t *ent, vec3_t mins, vec3_t maxs,
                         vec3_t offset) {
  qmodel_t *model;
  vec3_t size;
  vec3_t hullmins, hullmaxs;
  hull_t *hull;

  // decide which clipping hull to use, based on the size
  if (ent->solid == SOLID_BSP) {  // explicit hulls in the BSP model
    if (ent->movetype != MOVETYPE_PUSH)
      Go_Error("SOLID_BSP without MOVETYPE_PUSH");

    model = sv.models[(int)ent->modelindex];

    if (!model || model->type != mod_brush)
      Go_Error("MOVETYPE_PUSH with a non bsp model");

    VectorSubtract(maxs, mins, size);
    if (size[0] < 3)
      hull = &model->hulls[0];
    else if (size[0] <= 32)
      hull = &model->hulls[1];
    else
      hull = &model->hulls[2];

    // calculate an offset value to center the origin
    VectorSubtract(hull->clip_mins, mins, offset);
    VectorAdd(offset, ent->origin, offset);
  } else {  // create a temp hull from bounding box sizes

    VectorSubtract(ent->mins, maxs, hullmins);
    VectorSubtract(ent->maxs, mins, hullmaxs);
    hull = SV_HullForBox(hullmins, hullmaxs);

    VectorCopy(ent->origin, offset);
  }

  return hull;
}

/*
===============================================================================

ENTITY AREA CHECKING

===============================================================================
*/

typedef struct areanode_s {
  int axis;  // -1 = leaf node
  float dist;
  struct areanode_s *children[2];
  link_t trigger_edicts;
  link_t solid_edicts;
} areanode_t;

#define AREA_DEPTH 4
#define AREA_NODES 32

static areanode_t sv_areanodes[AREA_NODES];
static int sv_numareanodes;

/*
===============
SV_CreateAreaNode

===============
*/
areanode_t *SV_CreateAreaNode(int depth, vec3_t mins, vec3_t maxs) {
  areanode_t *anode;
  vec3_t size;
  vec3_t mins1, maxs1, mins2, maxs2;

  anode = &sv_areanodes[sv_numareanodes];
  sv_numareanodes++;

  ClearLink(&anode->trigger_edicts);
  ClearLink(&anode->solid_edicts);

  if (depth == AREA_DEPTH) {
    anode->axis = -1;
    anode->children[0] = anode->children[1] = NULL;
    return anode;
  }

  VectorSubtract(maxs, mins, size);
  if (size[0] > size[1])
    anode->axis = 0;
  else
    anode->axis = 1;

  anode->dist = 0.5 * (maxs[anode->axis] + mins[anode->axis]);
  VectorCopy(mins, mins1);
  VectorCopy(mins, mins2);
  VectorCopy(maxs, maxs1);
  VectorCopy(maxs, maxs2);

  maxs1[anode->axis] = mins2[anode->axis] = anode->dist;

  anode->children[0] = SV_CreateAreaNode(depth + 1, mins2, maxs2);
  anode->children[1] = SV_CreateAreaNode(depth + 1, mins1, maxs1);

  return anode;
}

/*
===============
SV_ClearWorld

===============
*/
void SV_ClearWorld(void) {
  SV_InitBoxHull();

  memset(sv_areanodes, 0, sizeof(sv_areanodes));
  sv_numareanodes = 0;
  SV_CreateAreaNode(0, sv.worldmodel->mins, sv.worldmodel->maxs);
}

/*
===============
SV_UnlinkEdict

===============
*/
static link_t **sv_link_next;
static link_t **sv_link_prev;

void SV_UnlinkEdict(int e) {
  // THERJAK
  edict_t *ent = EDICT_NUM(e);
  if (!ent->area2.prev) return;  // not linked in anywhere
  RemoveLink(&ent->area2);
  if (sv_link_next && *sv_link_next == &ent->area2)
    *sv_link_next = ent->area2.next;
  if (sv_link_prev && *sv_link_prev == &ent->area2)
    *sv_link_prev = ent->area2.prev;
  ent->area2.prev = ent->area2.next = NULL;
}

/*
====================
SV_TouchLinks
====================
*/
void SV_TouchLinks(int e, areanode_t *node) {
  link_t *l, *next;
  int touch;
  entvars_t *ev, *tv;
  int old_self, old_other;

  // touch linked edicts
  sv_link_next = &next;
  ev = EVars(e);
  for (l = node->trigger_edicts.next; l != &node->trigger_edicts; l = next) {
    if (!l) {
      // my area got removed out from under me!
      Con_Printf("SV_TouchLinks: encountered NULL link!\n");
      break;
    }

    next = l->next;
    touch = NUM_FOR_EDICT(EDICT_FROM_AREA(l));
    if (touch == e) continue;
    tv = EVars(touch);
    if (!tv->touch || tv->solid != SOLID_TRIGGER) continue;
    if (ev->absmin[0] > tv->absmax[0] || ev->absmin[1] > tv->absmax[1] ||
        ev->absmin[2] > tv->absmax[2] || ev->absmax[0] < tv->absmin[0] ||
        ev->absmax[1] < tv->absmin[1] || ev->absmax[2] < tv->absmin[2])
      continue;
    old_self = Pr_global_struct_self();
    old_other = Pr_global_struct_other();

    Set_pr_global_struct_self(touch);
    Set_pr_global_struct_other(e);
    Set_pr_global_struct_time(SV_Time());
    PR_ExecuteProgram(tv->touch);

    Set_pr_global_struct_self(old_self);
    Set_pr_global_struct_other(old_other);
  }

  sv_link_next = NULL;

  // recurse down both sides
  if (node->axis == -1) return;

  if (ev->absmax[node->axis] > node->dist) SV_TouchLinks(e, node->children[0]);
  if (ev->absmin[node->axis] < node->dist) SV_TouchLinks(e, node->children[1]);
}

/*
===============
SV_FindTouchedLeafs

===============
*/
void SV_FindTouchedLeafs(int e, mnode_t *node) {
  mplane_t *splitplane;
  mleaf_t *leaf;
  int sides;
  int leafnum;
  edict_t *ent = EDICT_NUM(e);
  entvars_t *ev;

  if (node->contents == CONTENTS_SOLID) return;

  // add an efrag if the node is a leaf

  if (node->contents < 0) {
    if (ent->num_leafs == MAX_ENT_LEAFS) return;

    leaf = (mleaf_t *)node;
    leafnum = leaf - sv.worldmodel->leafs - 1;

    ent->leafnums[ent->num_leafs] = leafnum;
    ent->num_leafs++;
    return;
  }

  // NODE_MIXED

  splitplane = node->plane;
  ev = EVars(e);
  sides = BOX_ON_PLANE_SIDE(ev->absmin, ev->absmax, splitplane);

  // recurse down the contacted sides
  if (sides & 1) SV_FindTouchedLeafs(e, node->children[0]);

  if (sides & 2) SV_FindTouchedLeafs(e, node->children[1]);
}

/*
===============
SV_LinkEdict

===============
*/
void SV_LinkEdict(int e, qboolean touch_triggers) {
  // THERJAK
  areanode_t *node;
  edict_t *ent = EDICT_NUM(e);
  entvars_t *ev;

  SV_UnlinkEdict(e);  // unlink from old position

  if (e == 0) return;  // don't add the world

  if (ent->free) return;

  ev = EVars(e);

  // set the abs box
  VectorAdd(ev->origin, ev->mins, ev->absmin);
  VectorAdd(ev->origin, ev->maxs, ev->absmax);

  //
  // to make items easier to pick up and allow them to be grabbed off
  // of shelves, the abs sizes are expanded
  //
  if ((int)ev->flags & FL_ITEM) {
    ev->absmin[0] -= 15;
    ev->absmin[1] -= 15;
    ev->absmax[0] += 15;
    ev->absmax[1] += 15;
  } else {  // because movement is clipped an epsilon away from an actual edge,
    // we must fully check even when bounding boxes don't quite touch
    ev->absmin[0] -= 1;
    ev->absmin[1] -= 1;
    ev->absmin[2] -= 1;
    ev->absmax[0] += 1;
    ev->absmax[1] += 1;
    ev->absmax[2] += 1;
  }

  // link to PVS leafs
  ent->num_leafs = 0;
  if (ev->modelindex) SV_FindTouchedLeafs(e, sv.worldmodel->nodes);

  if (ev->solid == SOLID_NOT) return;

  // find the first node that the ent's box crosses
  node = sv_areanodes;
  while (1) {
    if (node->axis == -1) break;
    if (ev->absmin[node->axis] > node->dist)
      node = node->children[0];
    else if (ev->absmax[node->axis] < node->dist)
      node = node->children[1];
    else
      break;  // crosses the node
  }

  // link it in

  if (ev->solid == SOLID_TRIGGER)
    InsertLinkBefore(&ent->area2, &node->trigger_edicts);
  else
    InsertLinkBefore(&ent->area2, &node->solid_edicts);

  // if touch_triggers, touch all entities at this node and decend for more
  if (touch_triggers) SV_TouchLinks(e, sv_areanodes);
}

/*
===============================================================================

POINT TESTING IN HULLS

===============================================================================
*/

/*
==================
SV_HullPointContents

==================
*/
int SV_HullPointContents(hull_t *hull, int num, vec3_t p) {
  float d;
  mclipnode_t *node;  // johnfitz -- was dclipnode_t
  mplane_t *plane;

  while (num >= 0) {
    if (num < hull->firstclipnode || num > hull->lastclipnode)
      Go_Error("SV_HullPointContents: bad node number");

    node = hull->clipnodes + num;
    plane = hull->planes + node->planenum;

    if (plane->type < 3)
      d = p[plane->type] - plane->dist;
    else
      d = DoublePrecisionDotProduct(plane->normal, p) - plane->dist;
    if (d < 0)
      num = node->children[1];
    else
      num = node->children[0];
  }

  return num;
}

/*
==================
SV_PointContents

==================
*/
int SV_PointContents(vec3_t p) {
  int cont;

  cont = SV_HullPointContents(&sv.worldmodel->hulls[0], 0, p);
  if (cont <= CONTENTS_CURRENT_0 && cont >= CONTENTS_CURRENT_DOWN)
    cont = CONTENTS_WATER;
  return cont;
}

//===========================================================================

/*
============
SV_TestEntityPosition

This could be a lot more efficient...
============
*/
qboolean SV_TestEntityPosition(int ent) {
  trace_t trace;
  entvars_t *ev;
  ev = EVars(ent);

  trace = SV_Move(ev->origin, ev->mins, ev->maxs, ev->origin, 0, ent);

  if (trace.startsolid) return true;

  return false;
}

/*
===============================================================================

LINE TESTING IN HULLS

===============================================================================
*/

/*
==================
SV_RecursiveHullCheck

==================
*/
qboolean SV_RecursiveHullCheck(hull_t *hull, int num, float p1f, float p2f,
                               vec3_t p1, vec3_t p2, trace_t *trace) {
  mclipnode_t *node;  // johnfitz -- was dclipnode_t
  mplane_t *plane;
  float t1, t2;
  float frac;
  int i;
  vec3_t mid;
  int side;
  float midf;

  // check for empty
  if (num < 0) {
    if (num != CONTENTS_SOLID) {
      trace->allsolid = false;
      if (num == CONTENTS_EMPTY)
        trace->inopen = true;
      else
        trace->inwater = true;
    } else
      trace->startsolid = true;
    return true;  // empty
  }

  if (num < hull->firstclipnode || num > hull->lastclipnode)
    Go_Error("SV_RecursiveHullCheck: bad node number");

  //
  // find the point distances
  //
  node = hull->clipnodes + num;
  plane = hull->planes + node->planenum;

  if (plane->type < 3) {
    t1 = p1[plane->type] - plane->dist;
    t2 = p2[plane->type] - plane->dist;
  } else {
    t1 = DoublePrecisionDotProduct(plane->normal, p1) - plane->dist;
    t2 = DoublePrecisionDotProduct(plane->normal, p2) - plane->dist;
  }

  if (t1 >= 0 && t2 >= 0)
    return SV_RecursiveHullCheck(hull, node->children[0], p1f, p2f, p1, p2,
                                 trace);
  if (t1 < 0 && t2 < 0)
    return SV_RecursiveHullCheck(hull, node->children[1], p1f, p2f, p1, p2,
                                 trace);

  // put the crosspoint DIST_EPSILON pixels on the near side
  if (t1 < 0)
    frac = (t1 + DIST_EPSILON) / (t1 - t2);
  else
    frac = (t1 - DIST_EPSILON) / (t1 - t2);
  if (frac < 0) frac = 0;
  if (frac > 1) frac = 1;

  midf = p1f + (p2f - p1f) * frac;
  for (i = 0; i < 3; i++) mid[i] = p1[i] + frac * (p2[i] - p1[i]);

  side = (t1 < 0);

  // move up to the node
  if (!SV_RecursiveHullCheck(hull, node->children[side], p1f, midf, p1, mid,
                             trace))
    return false;

#ifdef PARANOID
  if (SV_HullPointContents(sv_hullmodel, mid, node->children[side]) ==
      CONTENTS_SOLID) {
    Con_Printf("mid PointInHullSolid\n");
    return false;
  }
#endif

  if (SV_HullPointContents(hull, node->children[side ^ 1], mid) !=
      CONTENTS_SOLID)
    // go past the node
    return SV_RecursiveHullCheck(hull, node->children[side ^ 1], midf, p2f, mid,
                                 p2, trace);

  if (trace->allsolid) return false;  // never got out of the solid area

  //==================
  // the other side of the node is solid, this is the impact point
  //==================
  if (!side) {
    VectorCopy(plane->normal, trace->plane.normal);
    trace->plane.dist = plane->dist;
  } else {
    VectorSubtract(vec3_origin, plane->normal, trace->plane.normal);
    trace->plane.dist = -plane->dist;
  }

  while (SV_HullPointContents(hull, hull->firstclipnode, mid) ==
         CONTENTS_SOLID) {  // shouldn't really happen, but does occasionally
    frac -= 0.1;
    if (frac < 0) {
      trace->fraction = midf;
      VectorCopy(mid, trace->endpos);
      Con_DPrintf("backup past 0\n");
      return false;
    }
    midf = p1f + (p2f - p1f) * frac;
    for (i = 0; i < 3; i++) mid[i] = p1[i] + frac * (p2[i] - p1[i]);
  }

  trace->fraction = midf;
  VectorCopy(mid, trace->endpos);

  return false;
}

/*
==================
SV_ClipMoveToEntity

Handles selection or creation of a clipping hull, and offseting (and
eventually rotation) of the end points
==================
*/
trace_t SV_ClipMoveToEntity(int ent, vec3_t start, vec3_t mins, vec3_t maxs,
                            vec3_t end) {
  trace_t trace;
  vec3_t offset;
  vec3_t start_l, end_l;
  hull_t *hull;

  // fill in a default trace
  memset(&trace, 0, sizeof(trace_t));
  trace.fraction = 1;
  trace.allsolid = true;
  VectorCopy(end, trace.endpos);

  // get the clipping hull
  hull = SV_HullForEntity(EVars(ent), mins, maxs, offset);

  VectorSubtract(start, offset, start_l);
  VectorSubtract(end, offset, end_l);

  // trace a line through the apropriate clipping hull
  SV_RecursiveHullCheck(hull, hull->firstclipnode, 0, 1, start_l, end_l,
                        &trace);

  // fix trace up by the offset
  if (trace.fraction != 1) VectorAdd(trace.endpos, offset, trace.endpos);

  // did we clip the move?
  if (trace.fraction < 1 || trace.startsolid) {
    trace.entn = ent;
    trace.entp = true;
  }

  return trace;
}

//===========================================================================

/*
====================
SV_ClipToLinks

Mins and maxs enclose the entire area swept by the move
====================
*/
void SV_ClipToLinks(areanode_t *node, moveclip_t *clip) {
  link_t *l, *next;
  int touch;
  trace_t trace;
  entvars_t *tv;

  // touch linked edicts
  for (l = node->solid_edicts.next; l != &node->solid_edicts; l = next) {
    next = l->next;
    touch = NUM_FOR_EDICT(EDICT_FROM_AREA(l));
    tv = EVars(touch);
    if (tv->solid == SOLID_NOT) continue;
    if (touch == clip->passedict) continue;
    if (tv->solid == SOLID_TRIGGER) Go_Error("Trigger in clipping list");

    if (clip->type == MOVE_NOMONSTERS && tv->solid != SOLID_BSP) continue;

    if (clip->boxmins[0] > tv->absmax[0] || clip->boxmins[1] > tv->absmax[1] ||
        clip->boxmins[2] > tv->absmax[2] || clip->boxmaxs[0] < tv->absmin[0] ||
        clip->boxmaxs[1] < tv->absmin[1] || clip->boxmaxs[2] < tv->absmin[2])
      continue;

    if (clip->passedict >= 0 && EVars(clip->passedict)->size[0] && !tv->size[0])
      continue;  // points never interact

    // might intersect, so do an exact clip
    if (clip->trace.allsolid) return;
    if (clip->passedict >= 0) {
      if (tv->owner == clip->passedict)
        continue;  // don't clip against own missiles
      if (EVars(clip->passedict)->owner == touch)
        continue;  // don't clip against owner
    }

    if ((int)tv->flags & FL_MONSTER)
      trace = SV_ClipMoveToEntity(touch, clip->start, clip->mins2, clip->maxs2,
                                  clip->end);
    else
      trace = SV_ClipMoveToEntity(touch, clip->start, clip->mins, clip->maxs,
                                  clip->end);
    if (trace.allsolid || trace.startsolid ||
        trace.fraction < clip->trace.fraction) {
      trace.entn = touch;
      trace.entp = true;
      if (clip->trace.startsolid) {
        clip->trace = trace;
        clip->trace.startsolid = true;
      } else
        clip->trace = trace;
    } else if (trace.startsolid)
      clip->trace.startsolid = true;
  }

  // recurse down both sides
  if (node->axis == -1) return;

  if (clip->boxmaxs[node->axis] > node->dist)
    SV_ClipToLinks(node->children[0], clip);
  if (clip->boxmins[node->axis] < node->dist)
    SV_ClipToLinks(node->children[1], clip);
}

/*
==================
SV_MoveBounds
==================
*/
void SV_MoveBounds(vec3_t start, vec3_t mins, vec3_t maxs, vec3_t end,
                   vec3_t boxmins, vec3_t boxmaxs) {
  int i;

  for (i = 0; i < 3; i++) {
    if (end[i] > start[i]) {
      boxmins[i] = start[i] + mins[i] - 1;
      boxmaxs[i] = end[i] + maxs[i] + 1;
    } else {
      boxmins[i] = end[i] + mins[i] - 1;
      boxmaxs[i] = start[i] + maxs[i] + 1;
    }
  }
}

/*
==================
SV_Move
==================
*/
trace_t SV_Move(vec3_t start, vec3_t mins, vec3_t maxs, vec3_t end, int type,
                int passedict) {
  moveclip_t clip;
  int i;

  memset(&clip, 0, sizeof(moveclip_t));

  // clip to world
  clip.trace = SV_ClipMoveToEntity(0, start, mins, maxs, end);

  clip.start = start;
  clip.end = end;
  clip.mins = mins;
  clip.maxs = maxs;
  clip.type = type;
  clip.passedict = passedict;

  if (type == MOVE_MISSILE) {
    for (i = 0; i < 3; i++) {
      clip.mins2[i] = -15;
      clip.maxs2[i] = 15;
    }
  } else {
    VectorCopy(mins, clip.mins2);
    VectorCopy(maxs, clip.maxs2);
  }

  // create the bounding box of the entire move
  SV_MoveBounds(start, clip.mins2, clip.maxs2, end, clip.boxmins, clip.boxmaxs);

  // clip to entities
  SV_ClipToLinks(sv_areanodes, &clip);

  return clip.trace;
}
