// SPDX-License-Identifier: GPL-2.0-or-later
#ifndef __CVAR_H__
#define __CVAR_H__

typedef struct cvar_s {
  int id;
} cvar_t;

float Cvar_GetValue(cvar_t *variable);
const char *Cvar_GetString(cvar_t *variable);
void Cvar_FakeRegister(cvar_t *v, char *name);

#endif /* __CVAR_H__ */
