#include "quakedef.h"

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

void Cvar_SetQuick(cvar_t *var, char *value) { CvarSetQuick(var->id, value); }

void Cvar_SetValueQuick(cvar_t *var, float value) {
  CvarSetValueQuick(var->id, value);
}

void Cvar_FakeRegister(cvar_t *v, char *name) { v->id = CvarGetID(name); }

void Cvar_SetCallback(cvar_t *var, cvarcallback_t func) {
  CvarSetCallback(var->id, func);
}

void CallCvarCallback(int id, cvarcallback_t func) {
  cvar_t var;
  var.id = id;
  func(&var);
}
