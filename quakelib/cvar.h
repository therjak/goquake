#ifndef __CVAR_H__
#define __CVAR_H__

typedef struct cvar_s {
  int id;
} cvar_t;

typedef void (*cvarcallback_t)(struct cvar_s *);

float Cvar_GetValue(cvar_t *variable);
const char *Cvar_GetString(cvar_t *variable);
void Cvar_FakeRegister(cvar_t *v, char *name);
void Cvar_SetCallback(cvar_t *var, cvarcallback_t func);

#endif /* __CVAR_H__ */
