// SPDX-License-Identifier: GPL-2.0-or-later
// r_alias.c -- alias model rendering

#include "quakedef.h"

void R_DrawAliasModel(entity_t *e) {
  GL_BindBuffer(GL_ARRAY_BUFFER, e->model->meshvbo);
  GL_BindBuffer(GL_ELEMENT_ARRAY_BUFFER, e->model->meshindexesvbo);
}
