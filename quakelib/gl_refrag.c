// r_efrag.c

#include "quakedef.h"

efrag_t **lastlink;

vec3_t r_emins, r_emaxs;

/*
================
R_RemoveEfrags

Call when removing an object from the world or moving it to another position
================
*/
void R_RemoveEfrags(entity_t *ent) {
  efrag_t *ef, *old, *walk, **prev;

  ef = ent->efrag;

  while (ef) {
    prev = &ef->leaf->efrags;
    while (1) {
      walk = *prev;
      if (!walk) break;
      if (walk == ef) {  // remove this fragment
        *prev = ef->leafnext;
        break;
      } else
        prev = &walk->leafnext;
    }

    old = ef;
    ef = ef->entnext;

    // put it on the free list
    old->entnext = cl.free_efrags;
    cl.free_efrags = old;
  }

  ent->efrag = NULL;
}

/*
===================
R_SplitEntityOnNode
===================
*/
void R_SplitEntityOnNode(mnode_t *node, entity_t* r_addent) {
  efrag_t *ef;
  mplane_t *splitplane;
  mleaf_t *leaf;
  int sides;

  if (node->contents == CONTENTS_SOLID) {
    return;
  }

  // add an efrag if the node is a leaf

  if (node->contents < 0) {
    leaf = (mleaf_t *)node;

    // grab an efrag off the free list
    ef = cl.free_efrags;
    if (!ef) {
      return;  // no free fragments...
    }
    cl.free_efrags = cl.free_efrags->entnext;

    ef->entity = r_addent;

    // add the entity link
    *lastlink = ef;
    lastlink = &ef->entnext;
    ef->entnext = NULL;

    // set the leaf links
    ef->leaf = leaf;
    ef->leafnext = leaf->efrags;
    leaf->efrags = ef;

    return;
  }

  // NODE_MIXED

  splitplane = node->plane;
  sides = BOX_ON_PLANE_SIDE(r_emins, r_emaxs, splitplane);

  if (sides == 3) {
    // split on this plane
    // if this is the first splitter of this bmodel, remember it
  }

  // recurse down the contacted sides
  if (sides & 1) R_SplitEntityOnNode(node->children[0], r_addent);

  if (sides & 2) R_SplitEntityOnNode(node->children[1], r_addent);
}

/*
===========
R_AddEfrags
===========
*/
void R_AddEfrags(entity_t *ent) {
  qmodel_t *entmodel;
  int i;

  if (!ent->model) return;

  lastlink = &ent->efrag;
  entmodel = ent->model;

  for (i = 0; i < 3; i++) {
    r_emins[i] = ent->origin[i] + entmodel->mins[i];
    r_emaxs[i] = ent->origin[i] + entmodel->maxs[i];
  }

  R_SplitEntityOnNode(cl.worldmodel->nodes, ent);
}

/*
================
R_StoreEfrags -- johnfitz -- pointless switch statement removed.
================
*/
void R_StoreEfrags(efrag_t **ppefrag) {
  entity_t *pent;
  efrag_t *pefrag;

  while ((pefrag = *ppefrag) != NULL) {
    pent = pefrag->entity;

    if ((pent->visframe != R_framecount()) && (VisibleEntitiesNum() < MAX_VISEDICTS)) {
      AddVisibleEntity(pent);
      pent->visframe = R_framecount();
    }

    ppefrag = &pefrag->leafnext;
  }
}
