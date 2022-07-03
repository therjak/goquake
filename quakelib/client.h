// SPDX-License-Identifier: GPL-2.0-or-later
#ifndef _CLIENT_H_
#define _CLIENT_H_

// client_state_t should hold all pieces of the client state

typedef enum {
  ca_dedicated = 0,     // a dedicated server with no ability to start a client
  ca_disconnected = 1,  // full screen console with no connection
  ca_connected = 2      // valid netcon, talking to a server
} cactive_t;

#define MAX_TEMP_ENTITIES 256    // johnfitz -- was 64
#define MAX_STATIC_ENTITIES 512  // johnfitz -- was 128

#endif /* _CLIENT_H_ */
