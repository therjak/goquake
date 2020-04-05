/*
Copyright (C) 1996-2001 Id Software, Inc.
Copyright (C) 2002-2009 John Fitzgibbons and others
Copyright (C) 2010-2014 QuakeSpasm developers

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.

*/

#ifndef _Q_COMMON_H
#define _Q_COMMON_H

#include <stdarg.h>
#include <stdio.h>

#include "q_stdinc.h"

// comndef.h  -- general definitions

#if defined(_WIN32)
#ifdef _MSC_VER
#pragma warning(disable : 4244)
/* 'argument'	: conversion from 'type1' to 'type2',
                  possible loss of data */
#pragma warning(disable : 4305)
/* 'identifier'	: truncation from 'type1' to 'type2' */
/*  in our case, truncation from 'double' to 'float' */
#pragma warning(disable : 4267)
/* 'var'	: conversion from 'size_t' to 'type',
                  possible loss of data (/Wp64 warning) */
#endif /* _MSC_VER */
#endif /* _WIN32 */

#undef min
#undef max
#define q_min(a, b) (((a) < (b)) ? (a) : (b))
#define q_max(a, b) (((a) > (b)) ? (a) : (b))
#define CLAMP(_minval, x, _maxval) \
  ((x) < (_minval) ? (_minval) : (x) > (_maxval) ? (_maxval) : (x))

//============================================================================

void Q_memset(void *dest, int fill, size_t count);
void Q_memcpy(void *dest, const void *src, size_t count);

#include "strl_fn.h"

/* locale-insensitive strcasecmp replacement functions: */
extern int q_strncasecmp(const char *s1, const char *s2, size_t n);

/* snprintf, vsnprintf : always use our versions. */
extern int q_snprintf(char *str, size_t size, const char *format, ...)
    __attribute__((__format__(__printf__, 3, 4)));
extern int q_vsnprintf(char *str, size_t size, const char *format, va_list args)
    __attribute__((__format__(__printf__, 3, 0)));

//============================================================================

extern char com_token[1024];

const char *COM_Parse(const char *data);

const char *COM_SkipPath(const char *pathname);
void COM_StripExtension(const char *in, char *out, size_t outsize);
void COM_FileBase(const char *in, char *out, size_t outsize);
void COM_AddExtension(char *path, const char *extension, size_t len);
const char *COM_FileGetExtension(const char *in); /* doesn't return NULL */

//============================================================================

const char *Com_Gamedir();

#endif /* _Q_COMMON_H */
