#ifndef __CVAR_H__
#define __CVAR_H__

#define CVAR_NONE 0
#define CVAR_ARCHIVE (1U << 0)  // if set, causes it to be saved to config
#define CVAR_NOTIFY (1U << 1)
// changes will be broadcasted to all players (q1)
#define CVAR_SERVERINFO (1U << 2)
// added to serverinfo will be sent to clients (q1/net_dgrm.c and qwsv)

typedef struct cvar_s {
  int id;
} cvar_t;

typedef void (*cvarcallback_t)(struct cvar_s *);

float Cvar_GetValue(cvar_t *variable);
const char *Cvar_GetString(cvar_t *variable);

// registers a cvar
void Cvar_FakeRegister(cvar_t *v, char *name);

void Cvar_SetCallback(cvar_t *var, cvarcallback_t func);
// set a callback function to the var

void Cvar_Set(char *var_name, char *value);
// equivelant to "<name> <variable>" typed at the console

void Cvar_SetValue(char *var_name, float value);
// expands value to a string and calls Cvar_Set

void Cvar_SetQuick(cvar_t *var, char *value);
void Cvar_SetValueQuick(cvar_t *var, float value);
// these two accept a cvar pointer instead of a var name,
// but are otherwise identical to the "non-Quick" versions.
// the cvar MUST be registered.

void Cvar_Init(void);

#endif /* __CVAR_H__ */
