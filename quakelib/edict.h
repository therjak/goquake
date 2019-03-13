#ifndef _QUAKE_EDICT_H
#define _QUAKE_EDICT_H

#include "entity_state.h"

#define MAX_ENT_LEAFS 32
typedef struct edict_s {
  int free;
  // link_t area2; /* linked to a division node or leaf */

  int num_leafs;
  int leafnums[MAX_ENT_LEAFS];

  entity_state_t baseline;
  unsigned char alpha; /* johnfitz -- hack to support alpha since it's not part
                          of entvars_t */
  int sendinterval;    /* johnfitz -- send time until nextthink to client for
                               better lerp timing */

  float freetime; /* sv.time when the object was freed */
  // entvars_t vars; /* C exported fields from progs */

  /* other fields from progs come immediately after */
} edict_t;

edict_t *EDICT_NUM(int n);
int NUM_FOR_EDICT(edict_t *e);

#endif  // _QUAKE_EDICT_H
