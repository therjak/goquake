/* Locale insensitive ctype.h functions taken from the RPM library -
 * RPM is Copyright (c) 1998 by Red Hat Software, Inc.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or (at
 * your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 * See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA
 */

#ifndef Q_CTYPE_H
#define Q_CTYPE_H

static inline int q_isupper(int c) { return (c >= 'A' && c <= 'Z'); }

static inline int q_tolower(int c) {
  return ((q_isupper(c)) ? (c | ('a' - 'A')) : c);
}

#endif /* Q_CTYPE_H */
