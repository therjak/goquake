#include "arch_def.h"
#include "quakedef.h"

#define DEFAULT_MEMORY \
  (256 * 1024 * 1024)  // ericw -- was 72MB (64-bit) / 64MB (32-bit)

static quakeparms_t parms;

void Sys_Init() {
  host_parms = &parms;

  parms.memsize = DEFAULT_MEMORY;
  parms.membase = malloc(parms.memsize);

  if (!parms.membase) Go_Error("Not enough memory free; check disk space\n");
}


