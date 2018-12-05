#include "quakedef.h"

const char *Cvar_VariableString(char *var_name);
static cvar_t *cvar_vars;
static char cvar_null_string[] = "";

float Cvar_GetValue(cvar_t *variable) { return CvarGetValue(variable->id); }

const char *Cvar_GetString(cvar_t *variable) {
  static char buffer[2048];
  char *value = CvarGetString(variable->id);
  if (!value) {
    return cvar_null_string;
  }
  strncpy(buffer, value, 2048);
  free(value);
  return buffer;
}

const char *Cvar_GetName(cvar_t *variable) {
  static char buffer[2048];
  char *value = CvarGetName(variable->id);
  if (!value) {
    return cvar_null_string;
  }
  strncpy(buffer, value, 2048);
  free(value);
  return buffer;
}

void Cvar_Cycle_f(void) {
  int i;

  if (Cmd_Argc() < 3) {
    Con_Printf(
        "cycle <cvar> <value list>: cycle cvar through a list of values\n");
    return;
  }

  // loop through the args until you find one that matches the current cvar
  // value.
  // yes, this will get stuck on a list that contains the same value twice.
  // it's not worth dealing with, and i'm not even sure it can be dealt with.
  Sys_Print("Bad Cvar_Cycle");
  for (i = 2; i < Cmd_Argc(); i++) {
    // zero is assumed to be a string, even though it could actually be zero.
    // The worst case
    // is that the first time you call this command, it won't match on zero when
    // it should, but after that,
    // it will be comparing strings that all had the same source (the user) so
    // it will work.
    if (Cmd_ArgvAsDouble(i) == 0) {
      if (!strcmp(Cmd_Argv(i), Cvar_VariableString(Cmd_Argv(1)))) break;
    } else {
      if (Cmd_ArgvAsDouble(i) == Cvar_VariableValue(Cmd_Argv(1))) break;
    }
  }

  if (i == Cmd_Argc())
    Cvar_Set(Cmd_Argv(1), Cmd_Argv(2));  // no match
  else if (i + 1 == Cmd_Argc())
    Cvar_Set(Cmd_Argv(1), Cmd_Argv(2));  // matched last value in list
  else
    Cvar_Set(Cmd_Argv(1), Cmd_Argv(i + 1));  // matched earlier in list
}

void Cvar_Init(void) { Cmd_AddCommand("cycle", Cvar_Cycle_f); }

const char *Cvar_VariableString(char *var_name) {
  static char buffer[2048];
  char *value = CvarVariableString(var_name);
  if (!value) {
    return cvar_null_string;
  }
  strncpy(buffer, value, 2048);
  free(value);
  return buffer;
}

void Cvar_SetQuick(cvar_t *var, char *value) { CvarSetQuick(var->id, value); }

void Cvar_SetValueQuick(cvar_t *var, float value) {
  CvarSetValueQuick(var->id, value);
}

void Cvar_Register(cvar_t *v, char *name, char *string, int flags) {
  v->id = CvarRegister(name, string, flags);
}

void Cvar_FakeRegister(cvar_t *v, char *name) {
  v->id = CvarGetID(name);
}

void Cvar_SetCallback(cvar_t *var, cvarcallback_t func) {
  CvarSetCallback(var->id, func);
}

void CallCvarCallback(int id, cvarcallback_t func) {
  cvar_t var;
  var.id = id;
  func(&var);
}

void Cvar_WriteVariables(FILE *f) {
  // TODO(therjak)
  /*
  cvar_t *var;

  for (var = cvar_vars; var; var = var->next) {
    if (var->flags & CVAR_ARCHIVE)
      fprintf(f, "%s \"%s\"\n", Cvar_GetName(var), Cvar_GetString(var));
  }
  */
}
