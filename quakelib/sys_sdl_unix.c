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

int Sys_FileTime(const char *path) {
  FILE *f;

  f = fopen(path, "rb");

  if (f) {
    fclose(f);
    return 1;
  }

  return -1;
}

void Sys_mkdir(const char *path) {
  int rc = mkdir(path, 0777);
  if (rc != 0 && errno == EEXIST) {
    struct stat st;
    if (stat(path, &st) == 0 && S_ISDIR(st.st_mode)) rc = 0;
  }
  if (rc != 0) {
    rc = errno;
    Sys_Error("Unable to create directory %s: %s", path, strerror(rc));
  }
}

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

const char *Sys_ConsoleInput(void) {
  static char con_text[256];
  static int textlen;
  char c;
  fd_set set;
  struct timeval timeout;

  FD_ZERO(&set);
  FD_SET(0, &set);  // stdin
  timeout.tv_sec = 0;
  timeout.tv_usec = 0;

  while (select(1, &set, NULL, NULL, &timeout)) {
    read(0, &c, 1);
    if (c == '\n' || c == '\r') {
      con_text[textlen] = '\0';
      textlen = 0;
      return con_text;
    } else if (c == 8) {
      if (textlen) {
        textlen--;
        con_text[textlen] = '\0';
      }
      continue;
    }
    con_text[textlen] = c;
    textlen++;
    if (textlen < (int)sizeof(con_text))
      con_text[textlen] = '\0';
    else {
      // buffer is full
      textlen = 0;
      con_text[0] = '\0';
      Sys_Print("\nConsole input too long!\n");
      break;
    }
  }

  return NULL;
}
