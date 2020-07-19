// common.c -- misc functions used in client and server

#include "common.h"

#include <errno.h>

#include "q_ctype.h"
#include "quakedef.h"

// if a packfile directory differs from this, it is assumed to be hacked
#define PAK0_COUNT 339      /* id1/pak0.pak - v1.0x */
#define PAK0_CRC_V100 13900 /* id1/pak0.pak - v1.00 */
#define PAK0_CRC_V101 62751 /* id1/pak0.pak - v1.01 */
#define PAK0_CRC_V106 32981 /* id1/pak0.pak - v1.06 */
#define PAK0_CRC (PAK0_CRC_V106)
#define PAK0_COUNT_V091 308 /* id1/pak0.pak - v0.91/0.92, not supported */
#define PAK0_CRC_V091 28804 /* id1/pak0.pak - v0.91/0.92, not supported */

char com_token[1024];

/*

All of Quake's data access is through a hierchal file system, but the contents
of the file system can be transparently merged from several sources.

The "base directory" is the path to the directory holding the quake.exe and all
game directories.  The sys_* files pass this to host_init in
quakeparms_t->basedir.
This can be overridden with the "-basedir" command line parm to allow code
debugging in a different directory.  The base directory is only used during
filesystem initialization.

The "game directory" is the first tree on the search path and directory that all
generated files (savegames, screenshots, demos, config files) will be saved to.
This can be overridden with the "-game" command line parameter.  The game
directory can never be changed while quake is executing.  This is a precacution
against having a malicious server instruct clients to write files over areas
they
shouldn't.

The "cache directory" is only used during development to save network bandwidth,
especially over ISDN / T1 lines.  If there is a cache directory specified, when
a file is found by the normal search path, it will be mirrored into the cache
directory, then opened there.

FIXME:
The file "parms.txt" will be read out of the game directory and appended to the
current command line arguments to allow different games to initialize startup
parms differently.  This could be used to add a "-sspeed 22050" for the high
quality sound edition.  Because they are added at the end, they will not
override an explicit setting on the original command line.

*/

//============================================================================

/*
============================================================================

                                        LIBRARY REPLACEMENT FUNCTIONS

============================================================================
*/

int q_strncasecmp(const char *s1, const char *s2, size_t n) {
  const char *p1 = s1;
  const char *p2 = s2;
  char c1, c2;

  if (p1 == p2 || n == 0) return 0;

  do {
    c1 = q_tolower(*p1++);
    c2 = q_tolower(*p2++);
    if (c1 == '\0' || c1 != c2) break;
  } while (--n > 0);

  return (int)(c1 - c2);
}

/* platform dependant (v)snprintf function names: */
#if defined(_WIN32)
#define snprintf_func _snprintf
#define vsnprintf_func _vsnprintf
#else
#define snprintf_func snprintf
#define vsnprintf_func vsnprintf
#endif

int q_vsnprintf(char *str, size_t size, const char *format, va_list args) {
  int ret;

  ret = vsnprintf_func(str, size, format, args);

  if (ret < 0) ret = (int)size;
  if (size == 0) /* no buffer */
    return ret;
  if ((size_t)ret >= size) str[size - 1] = '\0';

  return ret;
}

int q_snprintf(char *str, size_t size, const char *format, ...) {
  int ret;
  va_list argptr;

  va_start(argptr, format);
  ret = q_vsnprintf(str, size, format, argptr);
  va_end(argptr);

  return ret;
}

void Q_memset(void *dest, int fill, size_t count) {
  size_t i;

  if ((((size_t)dest | count) & 3) == 0) {
    count >>= 2;
    fill = fill | (fill << 8) | (fill << 16) | (fill << 24);
    for (i = 0; i < count; i++) ((int *)dest)[i] = fill;
  } else
    for (i = 0; i < count; i++) ((byte *)dest)[i] = fill;
}

void Q_memcpy(void *dest, const void *src, size_t count) {
  size_t i;

  if ((((size_t)dest | (size_t)src | count) & 3) == 0) {
    count >>= 2;
    for (i = 0; i < count; i++) ((int *)dest)[i] = ((int *)src)[i];
  } else
    for (i = 0; i < count; i++) ((byte *)dest)[i] = ((byte *)src)[i];
}

/*
============
COM_SkipPath
============
*/

const char *COM_SkipPath(const char *pathname) {
  const char *last;

  last = pathname;
  while (*pathname) {
    if (*pathname == '/') last = pathname + 1;
    pathname++;
  }
  return last;
}

/*
============
COM_StripExtension
============
*/
void COM_StripExtension(const char *in, char *out, size_t outsize) {
  int length;

  if (!*in) {
    *out = '\0';
    return;
  }
  if (in != out) /* copy when not in-place editing */
    q_strlcpy(out, in, outsize);
  length = (int)strlen(out) - 1;
  while (length > 0 && out[length] != '.') {
    --length;
    if (out[length] == '/' || out[length] == '\\') return; /* no extension */
  }
  if (length > 0) out[length] = '\0';
}

/*
============
COM_FileBase
take 'somedir/otherdir/filename.ext',
write only 'filename' to the output
============
*/
void COM_FileBase(const char *in, char *out, size_t outsize) {
  const char *dot, *slash, *s;

  s = in;
  slash = in;
  dot = NULL;
  while (*s) {
    if (*s == '/') slash = s + 1;
    if (*s == '.') dot = s;
    s++;
  }
  if (dot == NULL) dot = s;

  if (dot - slash < 2)
    q_strlcpy(out, "?model?", outsize);
  else {
    size_t len = dot - slash;
    if (len >= outsize) len = outsize - 1;
    memcpy(out, slash, len);
    out[len] = '\0';
  }
}

/*
==============
COM_Parse

Parse a token out of a string
==============
*/
const char *COM_Parse(const char *data) {
  int c;
  int len;

  len = 0;
  com_token[0] = 0;

  if (!data) return NULL;

// skip whitespace
skipwhite:
  while ((c = *data) <= ' ') {
    if (c == 0) return NULL;  // end of file
    data++;
  }

  // skip // comments
  if (c == '/' && data[1] == '/') {
    while (*data && *data != '\n') data++;
    goto skipwhite;
  }

  // skip /*..*/ comments
  if (c == '/' && data[1] == '*') {
    data += 2;
    while (*data && !(*data == '*' && data[1] == '/')) data++;
    if (*data) data += 2;
    goto skipwhite;
  }

  // handle quoted strings specially
  if (c == '\"') {
    data++;
    while (1) {
      if ((c = *data) != 0) ++data;
      if (c == '\"' || !c) {
        com_token[len] = 0;
        return data;
      }
      com_token[len] = c;
      len++;
    }
  }

  // parse single characters
  if (c == '{' || c == '}' || c == '(' || c == ')' || c == '\'' || c == ':') {
    com_token[len] = c;
    len++;
    com_token[len] = 0;
    return data + 1;
  }

  // parse a regular word
  do {
    com_token[len] = c;
    data++;
    len++;
    c = *data;
    /* commented out the check for ':' so that ip:port works */
    if (c == '{' || c == '}' || c == '(' || c == ')' ||
        c == '\'' /* || c == ':' */)
      break;
  } while (c > 32);

  com_token[len] = 0;
  return data;
}
