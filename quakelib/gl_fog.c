// gl_fog.c -- global and volumetric fog

#include "quakedef.h"

//==============================================================================
//
//  GLOBAL FOG
//
//==============================================================================

#define DEFAULT_DENSITY 0.0
#define DEFAULT_GRAY 0.3

float fog_density;
float fog_red;
float fog_green;
float fog_blue;

float old_density;
float old_red;
float old_green;
float old_blue;

float fade_time;  // duration of fade
float fade_done;  // time when fade will be done

void Fog_Update(float density, float red, float green, float blue, float time) {
  // save previous settings for fade
  if (time > 0) {
    // check for a fade in progress
    if (fade_done > CL_Time()) {
      float f;

      f = (fade_done - CL_Time()) / fade_time;
      old_density = f * old_density + (1.0 - f) * fog_density;
      old_red = f * old_red + (1.0 - f) * fog_red;
      old_green = f * old_green + (1.0 - f) * fog_green;
      old_blue = f * old_blue + (1.0 - f) * fog_blue;
    } else {
      old_density = fog_density;
      old_red = fog_red;
      old_green = fog_green;
      old_blue = fog_blue;
    }
  }

  fog_density = density;
  fog_red = red;
  fog_green = green;
  fog_blue = blue;
  fade_time = time;
  fade_done = CL_Time() + time;
}

void Fog_FogCommand_f(void) {
  switch (Cmd_Argc()) {
    default:
    case 1:
      Con_Printf("usage:\n");
      Con_Printf("   fog <density>\n");
      Con_Printf("   fog <red> <green> <blue>\n");
      Con_Printf("   fog <density> <red> <green> <blue>\n");
      Con_Printf("current values:\n");
      Con_Printf("   \"density\" is \"%f\"\n", fog_density);
      Con_Printf("   \"red\" is \"%f\"\n", fog_red);
      Con_Printf("   \"green\" is \"%f\"\n", fog_green);
      Con_Printf("   \"blue\" is \"%f\"\n", fog_blue);
      break;
    case 2:
      Fog_Update(q_max(0.0, Cmd_ArgvAsDouble(1)), fog_red, fog_green, fog_blue,
                 0.0);
      break;
    case 3:  // TEST
      Fog_Update(q_max(0.0, Cmd_ArgvAsDouble(1)), fog_red, fog_green, fog_blue,
                 Cmd_ArgvAsDouble(2));
      break;
    case 4:
      Fog_Update(fog_density, CLAMP(0.0, Cmd_ArgvAsDouble(1), 1.0),
                 CLAMP(0.0, Cmd_ArgvAsDouble(2), 1.0),
                 CLAMP(0.0, Cmd_ArgvAsDouble(3), 1.0), 0.0);
      break;
    case 5:
      Fog_Update(q_max(0.0, Cmd_ArgvAsDouble(1)),
                 CLAMP(0.0, Cmd_ArgvAsDouble(2), 1.0),
                 CLAMP(0.0, Cmd_ArgvAsDouble(3), 1.0),
                 CLAMP(0.0, Cmd_ArgvAsDouble(4), 1.0), 0.0);
      break;
    case 6:  // TEST
      Fog_Update(q_max(0.0, Cmd_ArgvAsDouble(1)),
                 CLAMP(0.0, Cmd_ArgvAsDouble(2), 1.0),
                 CLAMP(0.0, Cmd_ArgvAsDouble(3), 1.0),
                 CLAMP(0.0, Cmd_ArgvAsDouble(4), 1.0), Cmd_ArgvAsDouble(5));
      break;
  }
}

void Fog_ParseWorldspawn(void) {
  char key[128], value[4096];
  const char *data;

  // initially no fog
  fog_density = DEFAULT_DENSITY;
  fog_red = DEFAULT_GRAY;
  fog_green = DEFAULT_GRAY;
  fog_blue = DEFAULT_GRAY;

  old_density = DEFAULT_DENSITY;
  old_red = DEFAULT_GRAY;
  old_green = DEFAULT_GRAY;
  old_blue = DEFAULT_GRAY;

  fade_time = 0.0;
  fade_done = 0.0;

  data = COM_Parse(cl.worldmodel->entities);
  if (!data) return;                // error
  if (com_token[0] != '{') return;  // error
  while (1) {
    data = COM_Parse(data);
    if (!data) return;               // error
    if (com_token[0] == '}') break;  // end of worldspawn
    if (com_token[0] == '_')
      strcpy(key, com_token + 1);
    else
      strcpy(key, com_token);
    while (key[strlen(key) - 1] == ' ')  // remove trailing spaces
      key[strlen(key) - 1] = 0;
    data = COM_Parse(data);
    if (!data) return;  // error
    strcpy(value, com_token);

    if (!strcmp("fog", key)) {
      // if found "fog" put the content of value into fog_X
      sscanf(value, "%f %f %f %f", &fog_density, &fog_red, &fog_green,
             &fog_blue);
    }
  }
}

float *Fog_GetColor(void) {
  static float c[4];
  float f;
  int i;

  if (fade_done > CL_Time()) {
    f = (fade_done - CL_Time()) / fade_time;
    c[0] = f * old_red + (1.0 - f) * fog_red;
    c[1] = f * old_green + (1.0 - f) * fog_green;
    c[2] = f * old_blue + (1.0 - f) * fog_blue;
    c[3] = 1.0;
  } else {
    c[0] = fog_red;
    c[1] = fog_green;
    c[2] = fog_blue;
    c[3] = 1.0;
  }

  // find closest 24-bit RGB value, so solid-colored sky can match the fog
  // perfectly
  for (i = 0; i < 3; i++) c[i] = (float)(Q_rint(c[i] * 255)) / 255.0f;

  return c;
}

float Fog_GetDensity(void) {
  float f;

  if (fade_done > CL_Time()) {
    f = (fade_done - CL_Time()) / fade_time;
    return f * old_density + (1.0 - f) * fog_density;
  } else
    return fog_density;
}

void Fog_SetupFrame(void) {
  glFogfv(GL_FOG_COLOR, Fog_GetColor());
  glFogf(GL_FOG_DENSITY, Fog_GetDensity() / 64.0);
}

void Fog_EnableGFog(void) {
  if (Fog_GetDensity() > 0) glEnable(GL_FOG);
}

void Fog_DisableGFog(void) {
  if (Fog_GetDensity() > 0) glDisable(GL_FOG);
}

void Fog_StartAdditive(void) {
  vec3_t color = {0, 0, 0};

  if (Fog_GetDensity() > 0) glFogfv(GL_FOG_COLOR, color);
}

void Fog_StopAdditive(void) {
  if (Fog_GetDensity() > 0) glFogfv(GL_FOG_COLOR, Fog_GetColor());
}

/*
void Fog_NewMap(void) {
  Fog_ParseWorldspawn();  // for global fog
}*/

void Fog_Init(void) {
  Cmd_AddCommand("fog", Fog_FogCommand_f);

  // set up global fog
  fog_density = DEFAULT_DENSITY;
  fog_red = DEFAULT_GRAY;
  fog_green = DEFAULT_GRAY;
  fog_blue = DEFAULT_GRAY;

  Fog_SetupState();
}

void Fog_SetupState(void) { glFogi(GL_FOG_MODE, GL_EXP2); }
