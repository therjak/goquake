/*
   Copyright (C) 1996-2001 Id Software, Inc.
   Copyright (C) 2010-2014 QuakeSpasm developers

   This program is free software; you can redistribute it and/or
   modify it under the terms of the GNU General Public License
   as published by the Free Software Foundation; either version 2
   of the License, or (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

   See the GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.

*/

#include "quakedef.h"
//
#include "_cgo_export.h"

typedef struct {
  int s;
  dfunction_t *f;
} prstack_t;

#define MAX_STACK_DEPTH 32
static prstack_t pr_stack[MAX_STACK_DEPTH];
static int pr_depth;

#define LOCALSTACK_SIZE 2048
static int localstack[LOCALSTACK_SIZE];
static int localstack_used;

qboolean pr_trace;
dfunction_t *pr_xfunction;
int pr_xstatement;
int pr_argc;

static const char *pr_opnames[] = {
    "DONE",

    "MUL_F",      "MUL_V",    "MUL_FV",   "MUL_VF",

    "DIV",

    "ADD_F",      "ADD_V",

    "SUB_F",      "SUB_V",

    "EQ_F",       "EQ_V",     "EQ_S",     "EQ_E",       "EQ_FNC",

    "NE_F",       "NE_V",     "NE_S",     "NE_E",       "NE_FNC",

    "LE",         "GE",       "LT",       "GT",

    "INDIRECT",   "INDIRECT", "INDIRECT", "INDIRECT",   "INDIRECT",
    "INDIRECT",

    "ADDRESS",

    "STORE_F",    "STORE_V",  "STORE_S",  "STORE_ENT",  "STORE_FLD",
    "STORE_FNC",

    "STOREP_F",   "STOREP_V", "STOREP_S", "STOREP_ENT", "STOREP_FLD",
    "STOREP_FNC",

    "RETURN",

    "NOT_F",      "NOT_V",    "NOT_S",    "NOT_ENT",    "NOT_FNC",

    "IF",         "IFNOT",

    "CALL0",      "CALL1",    "CALL2",    "CALL3",      "CALL4",
    "CALL5",      "CALL6",    "CALL7",    "CALL8",

    "STATE",

    "GOTO",

    "AND",        "OR",

    "BITAND",     "BITOR"};

const char *PR_GlobalString(int ofs);
const char *PR_GlobalStringNoContents(int ofs);

//=============================================================================

/*
   =================
   PR_PrintStatement
   =================
   */
static void PR_PrintStatement(dstatement_t *s) {
  int i;

  if ((unsigned int)s->op < sizeof(pr_opnames) / sizeof(pr_opnames[0])) {
    Con_Printf("%s ", pr_opnames[s->op]);
    i = strlen(pr_opnames[s->op]);
    for (; i < 10; i++) Con_Printf(" ");
  }

  if (s->op == OP_IF || s->op == OP_IFNOT)
    Con_Printf("%sbranch %i", PR_GlobalString(s->a), s->b);
  else if (s->op == OP_GOTO) {
    Con_Printf("branch %i", s->a);
  } else if ((unsigned int)(s->op - OP_STORE_F) < 6) {
    Con_Printf("%s", PR_GlobalString(s->a));
    Con_Printf("%s", PR_GlobalStringNoContents(s->b));
  } else {
    if (s->a) Con_Printf("%s", PR_GlobalString(s->a));
    if (s->b) Con_Printf("%s", PR_GlobalString(s->b));
    if (s->c) Con_Printf("%s", PR_GlobalStringNoContents(s->c));
  }
  Con_Printf("\n");
}

/*
   ============
   PR_StackTrace
   ============
   */
static void PR_StackTrace(void) {
  int i;
  dfunction_t *f;

  if (pr_depth == 0) {
    Con_Printf("<NO STACK>\n");
    return;
  }

  pr_stack[pr_depth].f = pr_xfunction;
  for (i = pr_depth; i >= 0; i--) {
    f = pr_stack[i].f;
    if (!f) {
      Con_Printf("<NO FUNCTION>\n");
    } else {
      Con_Printf("%12s : %s\n", PR_GetString(f->s_file),
                 PR_GetString(f->s_name));
    }
  }
}

/*
   ============
   PR_Profile_f

   ============
   */
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

/*
   ============
   PR_RunError

   Aborts the currently executing function
   ============
   */
void PR_RunError(const char *error, ...) {
  va_list argptr;
  char string[1024];

  va_start(argptr, error);
  q_vsnprintf(string, sizeof(string), error, argptr);
  va_end(argptr);

  PR_PrintStatement(pr_statements + pr_xstatement);
  PR_StackTrace();

  Con_Printf("%s\n", string);

  pr_depth = 0;  // dump the stack so host_error can shutdown functions

  Host_Error("Program error");
}

/*
   ====================
   PR_EnterFunction

   Returns the new program statement counter
   ====================
   */
static int PR_EnterFunction(dfunction_t *f) {
  int i, j, c, o;

  pr_stack[pr_depth].s = pr_xstatement;
  pr_stack[pr_depth].f = pr_xfunction;
  pr_depth++;
  if (pr_depth >= MAX_STACK_DEPTH) PR_RunError("stack overflow");

  // save off any locals that the new function steps on
  c = f->locals;
  if (localstack_used + c > LOCALSTACK_SIZE)
    PR_RunError("PR_ExecuteProgram: locals stack overflow\n");

  for (i = 0; i < c; i++) {
    localstack[localstack_used + i] = Pr_globalsi(f->parm_start + i);
    // localstack[localstack_used + i] = ((int *)pr_globals)[f->parm_start + i];
  }
  localstack_used += c;

  // copy parameters
  o = f->parm_start;
  for (i = 0; i < f->numparms; i++) {
    for (j = 0; j < f->parm_size[i]; j++) {
      Set_pr_globalsi(o, Pr_globalsi(OFS_PARM0 + i * 3 + j));
      // ((int *)pr_globals)[o] = ((int *)pr_globals)[OFS_PARM0 + i * 3 + j];
      o++;
    }
  }

  pr_xfunction = f;
  return f->first_statement - 1;  // offset the s++
}

/*
   ====================
   PR_LeaveFunction
   ====================
   */
static int PR_LeaveFunction(void) {
  int i, c;

  if (pr_depth <= 0) Host_Error("prog stack underflow");

  // Restore locals from the stack
  c = pr_xfunction->locals;
  localstack_used -= c;
  if (localstack_used < 0)
    PR_RunError("PR_ExecuteProgram: locals stack underflow");

  for (i = 0; i < c; i++) {
    Set_pr_globalsi(pr_xfunction->parm_start + i, localstack[localstack_used + i]);
    // ((int *)pr_globals)[pr_xfunction->parm_start + i] = localstack[localstack_used + i];
  }

  // up stack
  pr_depth--;
  pr_xfunction = pr_stack[pr_depth].f;
  return pr_stack[pr_depth].s;
}

/*
   ====================
   PR_ExecuteProgram

   The interpretation main loop
   ====================
   */
#define OPAF Pr_globalsf((unsigned short)st->a)
#define OPBF Pr_globalsf((unsigned short)st->b)
#define OPCF Pr_globalsf((unsigned short)st->c)

#define OPAI Pr_globalsi((unsigned short)st->a)
#define OPBI Pr_globalsi((unsigned short)st->b)
#define OPCI Pr_globalsi((unsigned short)st->c)

#define OPAV1 Pr_globalsf((unsigned short)st->a)
#define OPBV1 Pr_globalsf((unsigned short)st->b)
#define OPAV2 Pr_globalsf((unsigned short)st->a + 1)
#define OPBV2 Pr_globalsf((unsigned short)st->b + 1)
#define OPAV3 Pr_globalsf((unsigned short)st->a + 2)
#define OPBV3 Pr_globalsf((unsigned short)st->b + 2)

#define SOPBF(X) Set_pr_globalsf((unsigned short)st->b, X)
#define SOPCF(X) Set_pr_globalsf((unsigned short)st->c, X)

#define SOPBI(X) Set_pr_globalsi((unsigned short)st->b, X)
#define SOPCI(X) Set_pr_globalsi((unsigned short)st->c, X)

#define SOPBV1(X) Set_pr_globalsf((unsigned short)st->b, X)
#define SOPBV2(X) Set_pr_globalsf((unsigned short)st->b + 1, X)
#define SOPBV3(X) Set_pr_globalsf((unsigned short)st->b + 2, X)
#define SOPCV1(X) Set_pr_globalsf((unsigned short)st->c, X)
#define SOPCV2(X) Set_pr_globalsf((unsigned short)st->c + 1, X)
#define SOPCV3(X) Set_pr_globalsf((unsigned short)st->c + 2, X)

void PR_ExecuteProgram(GoInt32 fnum) {
  eval_t *ptr;
  dstatement_t *st;
  dfunction_t *f, *newf;
  int profile, startprofile;
  edict_t *ed;
  int exitdepth;

  if (!fnum || fnum >= progs->numfunctions) {
    if (Pr_global_struct_self()) ED_Print(EDICT_NUM(Pr_global_struct_self()));
    Host_Error("PR_ExecuteProgram: NULL function");
  }

  f = &pr_functions[fnum];

  pr_trace = false;

  // make a stack frame
  exitdepth = pr_depth;

  st = &pr_statements[PR_EnterFunction(f)];
  startprofile = profile = 0;

  while (1) {
    st++; /* next statement */

    if (++profile > 100000) {
      pr_xstatement = st - pr_statements;
      PR_RunError("runaway loop error");
    }

    if (pr_trace) PR_PrintStatement(st);

    switch (st->op) {
      case OP_ADD_F:
        SOPCF(OPAF + OPBF);
        // OPC->_float = OPA->_float + OPB->_float;
        break;
      case OP_ADD_V:
        SOPCV1(OPAV1 + OPBV1);
        SOPCV2(OPAV2 + OPBV2);
        SOPCV3(OPAV3 + OPBV3);
        // OPC->vector[0] = OPA->vector[0] + OPB->vector[0];
        // OPC->vector[1] = OPA->vector[1] + OPB->vector[1];
        // OPC->vector[2] = OPA->vector[2] + OPB->vector[2];
        break;

      case OP_SUB_F:
        SOPCF(OPAF - OPBF);
        // OPC->_float = OPA->_float - OPB->_float;
        break;
      case OP_SUB_V:
        SOPCV1(OPAV1 - OPBV1);
        SOPCV2(OPAV2 - OPBV2);
        SOPCV3(OPAV3 - OPBV3);
        // OPC->vector[0] = OPA->vector[0] - OPB->vector[0];
        // OPC->vector[1] = OPA->vector[1] - OPB->vector[1];
        // OPC->vector[2] = OPA->vector[2] - OPB->vector[2];
        break;

      case OP_MUL_F:
        SOPCF(OPAF * OPBF);
        // OPC->_float = OPA->_float * OPB->_float;
        break;
      case OP_MUL_V:
        SOPCF( OPAV1 * OPBV1 + OPAV2 * OPBV2 + OPAV3 * OPBV3);
//        OPC->_float = OPA->vector[0] * OPB->vector[0] +
//                      OPA->vector[1] * OPB->vector[1] +
//                      OPA->vector[2] * OPB->vector[2];
        break;
      case OP_MUL_FV:
        SOPCV1(OPAF * OPBV1);
        SOPCV2(OPAF * OPBV2);
        SOPCV3(OPAF * OPBV3);
        // OPC->vector[0] = OPA->_float * OPB->vector[0];
        // OPC->vector[1] = OPA->_float * OPB->vector[1];
        // OPC->vector[2] = OPA->_float * OPB->vector[2];
        break;
      case OP_MUL_VF:
        SOPCV1(OPBF * OPAV1);
        SOPCV2(OPBF * OPAV2);
        SOPCV3(OPBF * OPAV3);
        // OPC->vector[0] = OPB->_float * OPA->vector[0];
        // OPC->vector[1] = OPB->_float * OPA->vector[1];
        // OPC->vector[2] = OPB->_float * OPA->vector[2];
        break;

      case OP_DIV_F:
        SOPCF(OPAF / OPBF);
        // OPC->_float = OPA->_float / OPB->_float;
        break;

      case OP_BITAND:
        SOPCF((int)OPAF & (int)OPBF);
        //OPC->_float = (int)OPA->_float & (int)OPB->_float;
        break;

      case OP_BITOR:
        SOPCF((int)OPAF | (int)OPBF);
        //OPC->_float = (int)OPA->_float | (int)OPB->_float;
        break;

      case OP_GE:
        SOPCF(OPAF >= OPBF);
        // OPC->_float = OPA->_float >= OPB->_float;
        break;
      case OP_LE:
        SOPCF(OPAF <= OPBF);
        // OPC->_float = OPA->_float <= OPB->_float;
        break;
      case OP_GT:
        SOPCF(OPAF > OPBF);
        // OPC->_float = OPA->_float > OPB->_float;
        break;
      case OP_LT:
        SOPCF(OPAF < OPBF);
        // OPC->_float = OPA->_float < OPB->_float;
        break;
      case OP_AND:
        SOPCF(OPAF && OPBF);
        // OPC->_float = OPA->_float && OPB->_float;
        break;
      case OP_OR:
        SOPCF(OPAF || OPBF);
        // OPC->_float = OPA->_float || OPB->_float;
        break;

      case OP_NOT_F:
        SOPCF(!OPAF);
        // OPC->_float = !OPA->_float;
        break;
      case OP_NOT_V:
        SOPCF(!OPAV1 && !OPAV2 && !OPAV3);
        //OPC->_float = !OPA->vector[0] && !OPA->vector[1] && !OPA->vector[2];
        break;
      case OP_NOT_S:
        SOPCF(!OPAI || !*PR_GetString(OPAI));
        //OPC->_float = !OPA->string || !*PR_GetString(OPA->string);
        break;
      case OP_NOT_FNC:
        SOPCF(!OPAI);
        //OPC->_float = !OPA->function;
        break;
      case OP_NOT_ENT:
        SOPCF((EDICT_NUM(OPAI) == sv.edicts));
        //OPC->_float = (EDICT_NUM(OPA->edict) == sv.edicts);
        break;

      case OP_EQ_F:
        SOPCF(OPAF == OPBF);
        // OPC->_float = OPA->_float == OPB->_float;
        break;
      case OP_EQ_V:
        SOPCF((OPAV1 == OPBV1) && (OPAV2 == OPBV2) && (OPAV3 == OPBV3));
//        OPC->_float = (OPA->vector[0] == OPB->vector[0]) &&
//                      (OPA->vector[1] == OPB->vector[1]) &&
//                      (OPA->vector[2] == OPB->vector[2]);
        break;
      case OP_EQ_S:
        SOPCF(!strcmp(PR_GetString(OPAI), PR_GetString(OPBI)));
//        OPC->_float =
//            !strcmp(PR_GetString(OPA->string), PR_GetString(OPB->string));
        break;
      case OP_EQ_E:
        SOPCF(OPAI == OPBI);
        // OPC->_float = OPA->_int == OPB->_int;
        break;
      case OP_EQ_FNC:
        SOPCF(OPAI == OPBI);
        //OPC->_float = OPA->function == OPB->function;
        break;

      case OP_NE_F:
        SOPCF(OPAF != OPBF);
        // OPC->_float = OPA->_float != OPB->_float;
        break;
      case OP_NE_V:
        SOPCF((OPAV1 != OPBV1) || (OPAV2 != OPBV2) || (OPAV3 != OPBV3));
//        OPC->_float = (OPA->vector[0] != OPB->vector[0]) ||
//                      (OPA->vector[1] != OPB->vector[1]) ||
//                      (OPA->vector[2] != OPB->vector[2]);
        break;
      case OP_NE_S:
        SOPCF(strcmp(PR_GetString(OPAI), PR_GetString(OPBI)));
//        OPC->_float =
//            strcmp(PR_GetString(OPA->string), PR_GetString(OPB->string));
        break;
      case OP_NE_E:
        SOPCF(OPAI != OPBI);
        // OPC->_float = OPA->_int != OPB->_int;
        break;
      case OP_NE_FNC:
        SOPCF(OPAI != OPBI);
        //OPC->_float = OPA->function != OPB->function;
        break;

      case OP_STORE_F:
      case OP_STORE_ENT:
      case OP_STORE_FLD:  // integers
      case OP_STORE_S:
      case OP_STORE_FNC:  // pointers
        SOPBI(OPAI);
        // OPB->_int = OPA->_int;
        break;
      case OP_STORE_V:
        SOPBV1(OPAV1);
        SOPBV2(OPAV2);
        SOPBV3(OPAV3);
        // OPB->vector[0] = OPA->vector[0];
        // OPB->vector[1] = OPA->vector[1];
        // OPB->vector[2] = OPA->vector[2];
        break;

      case OP_STOREP_F:
      case OP_STOREP_ENT:
      case OP_STOREP_FLD:  // integers
      case OP_STOREP_S:
      case OP_STOREP_FNC:  // pointers
        ptr = (eval_t *)((byte *)EVars(0) + OPBI);
        ptr->_int = OPAI;
        // ptr = (eval_t *)((byte *)sv.edicts + OPB->_int);
        // ptr->_int = OPA->_int;
        break;
      case OP_STOREP_V:
        ptr = (eval_t *)((byte *)EVars(0) + OPBI);
        //ptr = (eval_t *)((byte *)sv.edicts + OPB->_int);
        ptr->vector[0] = OPAV1;
        ptr->vector[1] = OPAV2;
        ptr->vector[2] = OPAV3;
        // ptr->vector[0] = OPA->vector[0];
        // ptr->vector[1] = OPA->vector[1];
        // ptr->vector[2] = OPA->vector[2];
        break;

      case OP_ADDRESS:
        ed = EDICT_NUM(OPAI);
#ifdef PARANOID
        NUM_FOR_EDICT(ed);  // Make sure it's in range
#endif
        if (ed == (edict_t *)sv.edicts && sv.state == ss_active) {
          pr_xstatement = st - pr_statements;
          PR_RunError("assignment to world entity");
        }
        SOPCI((byte *)((int *)EVars(OPAI) + OPBI) - (byte *)EVars(0));
//        OPC->_int =
//            (byte *)((int *)EVars(OPA->edict) + OPB->_int) - (byte *)sv.edicts;
        break;

      case OP_LOAD_F:
      case OP_LOAD_FLD:
      case OP_LOAD_ENT:
      case OP_LOAD_S:
      case OP_LOAD_FNC:
        ed = EDICT_NUM(OPAI);
#ifdef PARANOID
        NUM_FOR_EDICT(ed);  // Make sure it's in range
#endif
        SOPCI(((eval_t *)((int *)EVars(OPAI) + OPBI))->_int);
//        OPC->_int = ((eval_t *)((int *)EVars(OPA->edict) + OPB->_int))->_int;
        break;

      case OP_LOAD_V:
        ed = EDICT_NUM(OPAI);
#ifdef PARANOID
        NUM_FOR_EDICT(ed);  // Make sure it's in range
#endif
        ptr = (eval_t *)((int *)EVars(OPAI) + OPBI);
        SOPCV1(ptr->vector[0]);
        SOPCV2(ptr->vector[1]);
        SOPCV3(ptr->vector[2]);
//        OPC->vector[0] = ptr->vector[0];
//        OPC->vector[1] = ptr->vector[1];
//        OPC->vector[2] = ptr->vector[2];
        break;

      case OP_IFNOT:
        if (!OPAI) st += st->b - 1; /* -1 to offset the st++ */
        break;

      case OP_IF:
        if (OPAI) st += st->b - 1; /* -1 to offset the st++ */
        break;

      case OP_GOTO:
        st += st->a - 1; /* -1 to offset the st++ */
        break;

      case OP_CALL0:
      case OP_CALL1:
      case OP_CALL2:
      case OP_CALL3:
      case OP_CALL4:
      case OP_CALL5:
      case OP_CALL6:
      case OP_CALL7:
      case OP_CALL8:
        pr_xfunction->profile += profile - startprofile;
        startprofile = profile;
        pr_xstatement = st - pr_statements;
        pr_argc = st->op - OP_CALL0;
        if (!OPAI) PR_RunError("NULL function");
        newf = &pr_functions[OPAI];
        if (newf->first_statement < 0) {  // Built-in function
          int i = -newf->first_statement;
          if (i >= pr_numbuiltins) PR_RunError("Bad builtin call number %d", i);
          pr_builtins[i]();
          break;
        }
        // Normal function
        st = &pr_statements[PR_EnterFunction(newf)];
        break;

      case OP_DONE:
      case OP_RETURN:
        pr_xfunction->profile += profile - startprofile;
        startprofile = profile;
        pr_xstatement = st - pr_statements;
        Set_pr_globalsf(OFS_RETURN, OPAV1);
        Set_pr_globalsf(OFS_RETURN + 1, OPAV2);
        Set_pr_globalsf(OFS_RETURN + 2, OPAV3);
        st = &pr_statements[PR_LeaveFunction()];
        if (pr_depth == exitdepth) {  // Done
          return;
        }
        break;

      case OP_STATE:
        ed = EDICT_NUM(Pr_global_struct_self());
        EVars(Pr_global_struct_self())->nextthink =
            Pr_global_struct_time() + 0.1;
        EVars(Pr_global_struct_self())->frame = OPAF;
        EVars(Pr_global_struct_self())->think = OPBI;
        break;

      default:
        pr_xstatement = st - pr_statements;
        PR_RunError("Bad opcode %i", st->op);
    }
  } /* end of while(1) loop */
}
