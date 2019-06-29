#include "quakedef.h"
//
#include "_cgo_export.h"

dfunction_t *pr_xfunction;

void PR_Profile_f(void) {
  int i, num;
  int pmax;
  dfunction_t *f, *best;

  if (!SV_Active()) return;

  num = 0;
  do {
    pmax = 0;
    best = NULL;
    for (i = 0; i < progs->numfunctions; i++) {
      f = &pr_functions[i];
      if (f->profile > pmax) {
        pmax = f->profile;
        best = f;
      }
    }
    if (best) {
      if (num < 10)
        Con_Printf("%7i %s\n", best->profile, PR_GetString(best->s_name));
      num++;
      best->profile = 0;
    }
  } while (best);
}
