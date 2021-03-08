// SPDX-License-Identifier: GPL-2.0-or-later
#include "arch_def.h"
//
#include "quakedef.h"
//
#include <errno.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <time.h>
#include <unistd.h>

#define DEFAULT_MEMORY \
  (256 * 1024 * 1024)  // ericw -- was 72MB (64-bit) / 64MB (32-bit)

static quakeparms_t parms;

void Sys_Init() {
  host_parms = &parms;

  parms.memsize = DEFAULT_MEMORY;
  parms.membase = malloc(parms.memsize);

  if (!parms.membase) Go_Error("Not enough memory free; check disk space\n");
}

void callQuakeFunc(xcommand_t f) { f(); }

void setInt(int* l, int v) { *l = v; }

static const char errortxt1[] = "\nERROR-OUT BEGIN\n\n";
static const char errortxt2[] = "\nQUAKE ERROR: ";

void Sys_Error(const char *error, ...) {
  va_list argptr;
  char text[1024];

  fputs(errortxt1, stderr);

  Host_Shutdown();

  va_start(argptr, error);
  q_vsnprintf(text, sizeof(text), error, argptr);
  va_end(argptr);

  fputs(errortxt2, stderr);
  fputs(text, stderr);
  fputs("\n\n", stderr);

  exit(1);
}
