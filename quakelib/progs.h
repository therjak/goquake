#ifndef _QUAKE_PROGS_H
#define _QUAKE_PROGS_H

#include "_cgo_export.h"

#include "pr_comp.h"  /* defs shared with qcc */
#include "progdefs.h" /* generated by program cdefs */

typedef union eval_s {
  GoInt32 string;
  float _float;
  float vector[3];
  GoInt32 function;
  int _int;
  int edict;
} eval_t;

#define MAX_ENT_LEAFS 32
typedef struct edict_s {
  qboolean free;
  link_t area; /* linked to a division node or leaf */

  int num_leafs;
  int leafnums[MAX_ENT_LEAFS];

  entity_state_t baseline;
  unsigned char alpha; /* johnfitz -- hack to support alpha since it's not part
                          of entvars_t */
  qboolean sendinterval; /* johnfitz -- send time until nextthink to client for
                            better lerp timing */

  float freetime; /* sv.time when the object was freed */
  // entvars_t vars; /* C exported fields from progs */

  /* other fields from progs come immediately after */
} edict_t;

entvars_t *EdictV(edict_t *e);

// FIXME: remove this mess!
#define EDICT_FROM_AREA(l) \
  ((edict_t *)((byte *)l - (intptr_t) & (((edict_t *)0)->area)))

//============================================================================

extern dprograms_t *progs;
extern dfunction_t *pr_functions;
extern dstatement_t *pr_statements;

void PR_Init(void);

void PR_ExecuteProgram(int fnum);
void PR_LoadProgs(void);

const char *PR_GetString(int num);
int PR_SetEngineString(char *s);

void PR_Profile_f(void);

void TT_ClearEdict(int e);

int ED_Alloc(void);
void ED_Free(int ed);

void ED_Write(FILE *f, int ed);
const char *ED_ParseEdict(const char *data, int ent);

void ED_WriteGlobals(FILE *f);
void ED_ParseGlobals(const char *data);

void ED_LoadFromFile(const char *data);

edict_t *AllocEdicts();
void FreeEdicts(edict_t *e);
edict_t *EDICT_NUM(int n);
int NUM_FOR_EDICT(edict_t *e);

extern int type_size[8];

typedef void (*builtin_t)(void);
extern builtin_t *pr_builtins;
extern int pr_numbuiltins;

extern int pr_argc;

extern qboolean pr_trace;
extern dfunction_t *pr_xfunction;
extern int pr_xstatement;

extern unsigned short pr_crc;

void PR_RunError(const char *error, ...);
//    __attribute__((__format__(__printf__, 1, 2), __noreturn__));

void ED_PrintEdicts(void);
void ED_PrintNum(int ent);

eval_t *GetEdictFieldValue(entvars_t *ev, const char *field);

#endif /* _QUAKE_PROGS_H */
