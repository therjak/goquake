#ifndef _CLIENT_H_
#define _CLIENT_H_

// client.h

typedef struct {
  char name[MAX_SCOREBOARDNAME];
} scoreboard_t;

#define NAME_LENGTH 64

//
// client_state_t should hold all pieces of the client state
//

#define SIGNONS 4  // signon messages to receive before connected

#define MAX_BEAMS 32  // johnfitz -- was 24
typedef struct {
  int entity;
  struct qmodel_s *model;
  float endtime;
  vec3_t start, end;
} beam_t;

#define MAX_EFRAGS 4096  // ericw -- was 2048 //johnfitz -- was 640

#define MAX_MAPSTRING 2048
#define MAX_DEMOS 8
#define MAX_DEMONAME 16

typedef enum {
  ca_dedicated = 0,     // a dedicated server with no ability to start a client
  ca_disconnected = 1,  // full screen console with no connection
  ca_connected = 2      // valid netcon, talking to a server
} cactive_t;

//
// the client_static_t structure is persistant through an arbitrary number
// of server connections
//
typedef struct {
  // demo loop control
  FILE *demofile;
} client_static_t;

extern client_static_t cls;

//
// the client_state_t structure is wiped completely at every
// server signon
//
typedef struct {
  //
  // information that is static for the entire time connected to a server
  //
  struct qmodel_s *model_precache[MAX_MODELS];

  char mapname[128];  // therjak

  // refresh related state
  struct qmodel_s *worldmodel;  // cl_entitites[0].model
  struct efrag_s *free_efrags;
  int num_entities;  // held in cl_entities array
  int num_statics;   // held in cl_staticentities array

  // frag scoreboard
  scoreboard_t *scores;  // [cl.maxclients] // therjak
} client_state_t;

extern entity_t cl_viewent;  // the gun model

extern client_state_t cl;
//
// cvars
//
extern cvar_t cl_shownet;

#define MAX_TEMP_ENTITIES 256    // johnfitz -- was 64
#define MAX_STATIC_ENTITIES 512  // johnfitz -- was 128
#define MAX_VISEDICTS 4096       // larger, now we support BSP2

// FIXME, allocate dynamically
extern efrag_t cl_efrags[MAX_EFRAGS];
extern entity_t cl_static_entities[MAX_STATIC_ENTITIES];
extern entity_t cl_temp_entities[MAX_TEMP_ENTITIES];
extern beam_t cl_beams[MAX_BEAMS];
extern entity_t *cl_visedicts[MAX_VISEDICTS];
extern int cl_numvisedicts;

extern entity_t *cl_entities;  // johnfitz -- was a static array, now on hunk

//=============================================================================

//
// cl_main
//

void CL_Init(void);

//
// cl_input
//
int CL_ReadFromServer(void);

void CL_UpdateTEnts(void);

void CL_ClearState(void);

//
// cl_demo.c
//
void CL_StopPlayback(void);
int CL_GetDemoMessage(void);

void CL_Stop_f(void);
void CL_Record_f(void);
void CL_PlayDemo_f(void);
void CL_TimeDemo_f(void);

//
// cl_parse.c
//
void CL_ParseServerMessage(void);

//
// cl_tent
//
void CL_InitTEnts(void);

//
// chase
//
extern cvar_t chase_active;

void Chase_Init(void);
void TraceLine(vec3_t start, vec3_t end, vec3_t impact);
void Chase_UpdateForDrawing(void);  // johnfitz

#endif /* _CLIENT_H_ */
