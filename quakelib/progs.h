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

//============================================================================

extern dprograms_t *progs;
extern dfunction_t *pr_functions;
extern dstatement_t *pr_statements;

void PR_ExecuteProgram(int fnum);
void PR_LoadProgs(void);

void PR_Profile_f(void);

void ED_Write(FILE *f, int ed);
const char *ED_ParseEdict(const char *data, int ent);

extern int type_size[8];

typedef void (*builtin_t)(void);
extern builtin_t *pr_builtins;
extern int pr_numbuiltins;

extern int pr_argc;

extern dfunction_t *pr_xfunction;

void PR_RunError(const char *error, ...);
//    __attribute__((__format__(__printf__, 1, 2), __noreturn__));

#endif /* _QUAKE_PROGS_H */
