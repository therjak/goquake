// chase.c -- chase camera code

#include "quakedef.h"

cvar_t chase_active;

/*
==============
Chase_Init
==============
*/
void Chase_Init(void) {
  Cvar_FakeRegister(&chase_active, "chase_active");
}
