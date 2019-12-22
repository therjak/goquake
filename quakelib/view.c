// view.c -- player eye positioning

#include "quakedef.h"

/*

The view is allowed to move slightly from it's true position for bobbing,
but if it exceeds 8 pixels linear distance (spherical, not box), the list of
entities sent from the server may not include everything in the pvs, especially
when crossing a water boudnary.

*/

void SetCLWeaponModel(int v) {
  entity_t *view;
  view = &cl_viewent;
  view->model = cl.model_precache[v];
}
