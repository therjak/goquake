/*
Copyright (C) 1996-2001 Id Software, Inc.
Copyright (C) 2002-2009 John Fitzgibbons and others
Copyright (C) 2009-2010 Ozkan Sezer
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

/*
        net.h
        quake's interface to the networking layer
        network functions and data, common to the
        whole engine
*/

#ifndef _QUAKE_NET_H
#define _QUAKE_NET_H

#define NET_NAMELEN 64

#define NET_MAXMESSAGE 64000 /* ericw -- was 32000 */

extern int DEFAULTnet_hostport;
extern int net_hostport;

extern cvar_t hostname;

void NET_Init(void);
void NET_Shutdown(void);

int NET_CheckNewConnections(void);
// returns a new connection number if there is one pending, else -1

int NET_Connect(char *host);
// called by client to connect to a host.  Returns -1 if not able to

const char *NET_QSocketGetAddressString(int sock);

void NET_Poll(void);

// Server list related globals:
extern qboolean slistInProgress;
extern qboolean slistSilent;
extern qboolean slistLocal;

extern int hostCacheCount;

void NET_Slist_f(void);
void NET_SlistSort(void);
const char *NET_SlistPrintServer(int n);
const char *NET_SlistPrintServerName(int n);

/* FIXME: driver related, but public:
 */
extern char my_tcpip_address[NET_NAMELEN];

#endif /* _QUAKE_NET_H */
