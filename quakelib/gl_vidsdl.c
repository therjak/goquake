#include <SDL2/SDL.h>
#include "quakedef.h"
#include "resource.h"

#define MAX_MODE_LIST 600
#define MAX_BPPS_LIST 5
#define WARP_WIDTH 320
#define WARP_HEIGHT 200
#define MAXWIDTH 10000
#define MAXHEIGHT 10000

#define DEFAULT_SDL_FLAGS SDL_OPENGL

typedef struct {
  int width;
  int height;
  int bpp;
} vmode_t;

static char *gl_extensions_nice;

static void GL_Init(void);
static void GL_SetupState(void);

qboolean gl_mtexable = false;
qboolean gl_texture_env_combine = false;
qboolean gl_texture_env_add = false;
qboolean gl_anisotropy_able = false;
float gl_max_anisotropy;
qboolean gl_vbo_able = false;
qboolean gl_glsl_able = false;
GLint gl_max_texture_units = 0;
int gl_stencilbits;

PFNGLMULTITEXCOORD2FARBPROC GL_MTexCoord2fFunc = NULL;
PFNGLACTIVETEXTUREARBPROC GL_SelectTextureFunc = NULL;
PFNGLCLIENTACTIVETEXTUREARBPROC GL_ClientActiveTextureFunc = NULL;
PFNGLBINDBUFFERARBPROC GL_BindBufferFunc = NULL;
PFNGLBUFFERDATAARBPROC GL_BufferDataFunc = NULL;
PFNGLBUFFERSUBDATAARBPROC GL_BufferSubDataFunc = NULL;
PFNGLDELETEBUFFERSARBPROC GL_DeleteBuffersFunc = NULL;
PFNGLGENBUFFERSARBPROC GL_GenBuffersFunc = NULL;

QS_PFNGLCREATESHADERPROC GL_CreateShaderFunc = NULL;
QS_PFNGLDELETESHADERPROC GL_DeleteShaderFunc = NULL;
QS_PFNGLDELETEPROGRAMPROC GL_DeleteProgramFunc = NULL;
QS_PFNGLSHADERSOURCEPROC GL_ShaderSourceFunc = NULL;
QS_PFNGLCOMPILESHADERPROC GL_CompileShaderFunc = NULL;
QS_PFNGLGETSHADERIVPROC GL_GetShaderivFunc = NULL;
QS_PFNGLGETSHADERINFOLOGPROC GL_GetShaderInfoLogFunc = NULL;
QS_PFNGLGETPROGRAMIVPROC GL_GetProgramivFunc = NULL;
QS_PFNGLGETPROGRAMINFOLOGPROC GL_GetProgramInfoLogFunc = NULL;
QS_PFNGLCREATEPROGRAMPROC GL_CreateProgramFunc = NULL;
QS_PFNGLATTACHSHADERPROC GL_AttachShaderFunc = NULL;
QS_PFNGLLINKPROGRAMPROC GL_LinkProgramFunc = NULL;
QS_PFNGLBINDATTRIBLOCATIONFUNC GL_BindAttribLocationFunc = NULL;
QS_PFNGLUSEPROGRAMPROC GL_UseProgramFunc = NULL;
QS_PFNGLGETATTRIBLOCATIONPROC GL_GetAttribLocationFunc = NULL;
QS_PFNGLVERTEXATTRIBPOINTERPROC GL_VertexAttribPointerFunc = NULL;
QS_PFNGLENABLEVERTEXATTRIBARRAYPROC GL_EnableVertexAttribArrayFunc = NULL;
QS_PFNGLDISABLEVERTEXATTRIBARRAYPROC GL_DisableVertexAttribArrayFunc = NULL;
QS_PFNGLGETUNIFORMLOCATIONPROC GL_GetUniformLocationFunc = NULL;
QS_PFNGLUNIFORM1IPROC GL_Uniform1iFunc = NULL;
QS_PFNGLUNIFORM1FPROC GL_Uniform1fFunc = NULL;
QS_PFNGLUNIFORM3FPROC GL_Uniform3fFunc = NULL;
QS_PFNGLUNIFORM4FPROC GL_Uniform4fFunc = NULL;

//====================================

static cvar_t vid_fullscreen;
static cvar_t vid_width;
static cvar_t vid_height;
static cvar_t vid_bpp;
static cvar_t vid_vsync;
static cvar_t vid_desktopfullscreen;
static cvar_t vid_borderless;

cvar_t vid_gamma;
cvar_t vid_contrast;

static void VID_Restart(void) {
  int width, height, bpp;
  qboolean fullscreen;

  if (VID_Locked() || !VIDChanged()) return;

  width = (int)Cvar_GetValue(&vid_width);
  height = (int)Cvar_GetValue(&vid_height);
  bpp = (int)Cvar_GetValue(&vid_bpp);
  fullscreen = Cvar_GetValue(&vid_fullscreen) ? true : false;

  //
  // validate new mode
  //
  if (!VID_ValidMode(width, height, bpp, fullscreen)) {
    Sys_Print("VID_ValidMode == false");
    Con_Printf("%dx%dx%d %s is not a valid mode\n", width, height, bpp,
               fullscreen ? "fullscreen" : "windowed");
    return;
  }

  // ericw -- OS X, SDL1: textures, VBO's invalid after mode change
  //          OS X, SDL2: still valid after mode change
  // To handle both cases, delete all GL objects (textures, VBO, GLSL) now.
  // We must not interleave deleting the old objects with creating new ones,
  // because
  // one of the new objects could be given the same ID as an invalid handle
  // which is later deleted.

  TexMgr_DeleteTextureObjects();
  GLSLGamma_DeleteTexture();
  R_DeleteShaders();
  GL_DeleteBModelVertexBuffer();
  GLMesh_DeleteVertexBuffers();

  //
  // set new mode
  //
  VID_SetMode(width, height, bpp, fullscreen);

  GL_Init();
  TexMgr_ReloadImages();
  GL_BuildBModelVertexBuffer();
  GLMesh_LoadVertexBuffers();
  GL_SetupState();
  Fog_SetupState();

  // warpimages needs to be recalculated
  TexMgr_RecalcWarpImageSize();

  UpdateConsoleSize();
  //
  // keep cvars in line with actual mode
  //
  VID_SyncCvars();
  //
  // update mouse grab
  //
  if (GetKeyDest() == key_console || GetKeyDest() == key_menu) {
    if (VID_GetModeState() == MS_WINDOWED)
      IN_Deactivate();
    else if (VID_GetModeState() == MS_FULLSCREEN)
      IN_Activate();
  }
}

static void VID_Test(void) {
  int old_width, old_height, old_bpp, old_fullscreen;

  if (VID_Locked() || !VIDChanged()) return;
  old_width = VID_GetCurrentWidth();
  old_height = VID_GetCurrentHeight();
  old_bpp = VID_GetCurrentBPP();
  old_fullscreen = VID_GetFullscreen() ? true : false;

  VID_Restart();

  if (!SCR_ModalMessage("Would you like to keep this\nvideo mode? (y/n)\n",
                        5.0f)) {
    Cvar_SetValueQuick(&vid_width, old_width);
    Cvar_SetValueQuick(&vid_height, old_height);
    Cvar_SetValueQuick(&vid_bpp, old_bpp);
    Cvar_SetQuick(&vid_fullscreen, old_fullscreen ? "1" : "0");
    VID_Restart();
  }
}

static qboolean GL_ParseExtensionList(const char *list, const char *name) {
  const char *start;
  const char *where, *terminator;

  if (!list || !name || !*name) return false;
  if (strchr(name, ' ') != NULL)
    return false;  // extension names must not have spaces

  start = list;
  while (1) {
    where = strstr(start, name);
    if (!where) break;
    terminator = where + strlen(name);
    if (where == start || where[-1] == ' ')
      if (*terminator == ' ' || *terminator == '\0') return true;
    start = terminator;
  }
  return false;
}

static void GL_CheckExtensions(const char *gl_extensions) {
  int swap_control;

  // ARB_vertex_buffer_object
  //
  GL_BindBufferFunc =
      (PFNGLBINDBUFFERARBPROC)GO_GL_GetProcAddress("glBindBufferARB");
  // 2.1
  GL_BufferDataFunc =
      (PFNGLBUFFERDATAARBPROC)GO_GL_GetProcAddress("glBufferDataARB");
  // 2.1
  GL_BufferSubDataFunc =
      (PFNGLBUFFERSUBDATAARBPROC)GO_GL_GetProcAddress("glBufferSubDataARB");
  // 2.1
  GL_DeleteBuffersFunc =
      (PFNGLDELETEBUFFERSARBPROC)GO_GL_GetProcAddress("glDeleteBuffersARB");
  // 2.1
  GL_GenBuffersFunc =
      (PFNGLGENBUFFERSARBPROC)GO_GL_GetProcAddress("glGenBuffersARB");
  // 2.1
  if (GL_BindBufferFunc && GL_BufferDataFunc && GL_BufferSubDataFunc &&
      GL_DeleteBuffersFunc && GL_GenBuffersFunc) {
    Con_Printf("FOUND: ARB_vertex_buffer_object\n");
    gl_vbo_able = true;
  } else {
    Con_Warning("ARB_vertex_buffer_object not available\n");
  }

  // multitexture
  //
  if (!CMLMtext())
    Con_Warning("Mutitexture disabled at command line\n");
  else if (GL_ParseExtensionList(gl_extensions, "GL_ARB_multitexture")) {
    GL_MTexCoord2fFunc = (PFNGLMULTITEXCOORD2FARBPROC)GO_GL_GetProcAddress(
        "glMultiTexCoord2fARB");
    // 2.1
    GL_SelectTextureFunc =
        (PFNGLACTIVETEXTUREARBPROC)GO_GL_GetProcAddress("glActiveTextureARB");
    // 2.1
    GL_ClientActiveTextureFunc =
        (PFNGLCLIENTACTIVETEXTUREARBPROC)GO_GL_GetProcAddress(
            "glClientActiveTextureARB");
    if (GL_MTexCoord2fFunc && GL_SelectTextureFunc &&
        GL_ClientActiveTextureFunc) {
      Con_Printf("FOUND: ARB_multitexture\n");
      gl_mtexable = true;

      glGetIntegerv(GL_MAX_TEXTURE_UNITS, &gl_max_texture_units);
      Con_Printf("GL_MAX_TEXTURE_UNITS: %d\n", (int)gl_max_texture_units);
    } else {
      Con_Warning("Couldn't link to multitexture functions\n");
    }
  } else {
    Con_Warning("multitexture not supported (extension not found)\n");
  }

  // texture_env_combine
  //
  if (!CMLCombine())
    Con_Warning("texture_env_combine disabled at command line\n");
  else if (GL_ParseExtensionList(gl_extensions, "GL_ARB_texture_env_combine")) {
    Con_Printf("FOUND: ARB_texture_env_combine\n");
    gl_texture_env_combine = true;
  } else if (GL_ParseExtensionList(gl_extensions,
                                   "GL_EXT_texture_env_combine")) {
    Con_Printf("FOUND: EXT_texture_env_combine\n");
    gl_texture_env_combine = true;
  } else {
    Con_Warning("texture_env_combine not supported\n");
  }

  // texture_env_add
  //
  if (!CMLAdd())
    Con_Warning("texture_env_add disabled at command line\n");
  else if (GL_ParseExtensionList(gl_extensions, "GL_ARB_texture_env_add")) {
    Con_Printf("FOUND: ARB_texture_env_add\n");
    gl_texture_env_add = true;
  } else if (GL_ParseExtensionList(gl_extensions, "GL_EXT_texture_env_add")) {
    Con_Printf("FOUND: EXT_texture_env_add\n");
    gl_texture_env_add = true;
  } else {
    Con_Warning("texture_env_add not supported\n");
  }

  // swap control
  //
  if (!VIDGLSwapControl()) {
    Con_Warning(
        "vertical sync not supported (SDL_GL_SetSwapInterval failed)\n");
  } else if ((swap_control = SDL_GL_GetSwapInterval()) == -1) {
    SetVIDGLSwapControl(false);
    Con_Warning(
        "vertical sync not supported (SDL_GL_GetSwapInterval failed)\n");
  } else if ((Cvar_GetValue(&vid_vsync) && swap_control != 1) ||
             (!Cvar_GetValue(&vid_vsync) && swap_control != 0)) {
    SetVIDGLSwapControl(false);
    Con_Warning(
        "vertical sync not supported (swap_control doesn't match vid_vsync)\n");
  } else {
    Con_Printf("FOUND: SDL_GL_SetSwapInterval\n");
  }

  // anisotropic filtering
  //
  if (GL_ParseExtensionList(gl_extensions,
                            "GL_EXT_texture_filter_anisotropic")) {
    float test1, test2;
    GLuint tex;

    // test to make sure we really have control over it
    // 1.0 and 2.0 should always be legal values
    glGenTextures(1, &tex);
    glBindTexture(GL_TEXTURE_2D, tex);
    glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_MAX_ANISOTROPY_EXT, 1.0f);
    glGetTexParameterfv(GL_TEXTURE_2D, GL_TEXTURE_MAX_ANISOTROPY_EXT, &test1);
    glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_MAX_ANISOTROPY_EXT, 2.0f);
    glGetTexParameterfv(GL_TEXTURE_2D, GL_TEXTURE_MAX_ANISOTROPY_EXT, &test2);
    glDeleteTextures(1, &tex);

    if (test1 == 1 && test2 == 2) {
      Con_Printf("FOUND: EXT_texture_filter_anisotropic\n");
      gl_anisotropy_able = true;
    } else {
      Con_Warning(
          "anisotropic filtering locked by driver. Current driver setting is "
          "%f\n",
          test1);
    }

    // get max value either way, so the menu and stuff know it
    glGetFloatv(GL_MAX_TEXTURE_MAX_ANISOTROPY_EXT, &gl_max_anisotropy);
    if (gl_max_anisotropy < 2) {
      gl_anisotropy_able = false;
      gl_max_anisotropy = 1;
      Con_Warning("anisotropic filtering broken: disabled\n");
    }
  } else {
    gl_max_anisotropy = 1;
    Con_Warning("texture_filter_anisotropic not supported\n");
  }

  // texture_non_power_of_two
  //
  if (GL_ParseExtensionList(gl_extensions, "GL_ARB_texture_non_power_of_two")) {
    Con_Printf("FOUND: ARB_texture_non_power_of_two\n");
  } else {
    Go_Error("texture_non_power_of_two not supported\n");
  }

  // GLSL
  //
  GL_CreateShaderFunc =
      (QS_PFNGLCREATESHADERPROC)GO_GL_GetProcAddress("glCreateShader");
  GL_DeleteShaderFunc =
      (QS_PFNGLDELETESHADERPROC)GO_GL_GetProcAddress("glDeleteShader");
  GL_DeleteProgramFunc =
      (QS_PFNGLDELETEPROGRAMPROC)GO_GL_GetProcAddress("glDeleteProgram");
  GL_ShaderSourceFunc =
      (QS_PFNGLSHADERSOURCEPROC)GO_GL_GetProcAddress("glShaderSource");
  GL_CompileShaderFunc =
      (QS_PFNGLCOMPILESHADERPROC)GO_GL_GetProcAddress("glCompileShader");
  GL_GetShaderivFunc =
      (QS_PFNGLGETSHADERIVPROC)GO_GL_GetProcAddress("glGetShaderiv");
  GL_GetShaderInfoLogFunc =
      (QS_PFNGLGETSHADERINFOLOGPROC)GO_GL_GetProcAddress("glGetShaderInfoLog");
  GL_GetProgramivFunc =
      (QS_PFNGLGETPROGRAMIVPROC)GO_GL_GetProcAddress("glGetProgramiv");
  GL_GetProgramInfoLogFunc =
      (QS_PFNGLGETPROGRAMINFOLOGPROC)GO_GL_GetProcAddress(
          "glGetProgramInfoLog");
  GL_CreateProgramFunc =
      (QS_PFNGLCREATEPROGRAMPROC)GO_GL_GetProcAddress("glCreateProgram");
  GL_AttachShaderFunc =
      (QS_PFNGLATTACHSHADERPROC)GO_GL_GetProcAddress("glAttachShader");
  GL_LinkProgramFunc =
      (QS_PFNGLLINKPROGRAMPROC)GO_GL_GetProcAddress("glLinkProgram");
  GL_BindAttribLocationFunc =
      (QS_PFNGLBINDATTRIBLOCATIONFUNC)GO_GL_GetProcAddress(
          "glBindAttribLocation");
  GL_UseProgramFunc =
      (QS_PFNGLUSEPROGRAMPROC)GO_GL_GetProcAddress("glUseProgram");
  GL_GetAttribLocationFunc =
      (QS_PFNGLGETATTRIBLOCATIONPROC)GO_GL_GetProcAddress(
          "glGetAttribLocation");
  GL_VertexAttribPointerFunc =
      (QS_PFNGLVERTEXATTRIBPOINTERPROC)GO_GL_GetProcAddress(
          "glVertexAttribPointer");
  GL_EnableVertexAttribArrayFunc =
      (QS_PFNGLENABLEVERTEXATTRIBARRAYPROC)GO_GL_GetProcAddress(
          "glEnableVertexAttribArray");
  GL_DisableVertexAttribArrayFunc =
      (QS_PFNGLDISABLEVERTEXATTRIBARRAYPROC)GO_GL_GetProcAddress(
          "glDisableVertexAttribArray");
  GL_GetUniformLocationFunc =
      (QS_PFNGLGETUNIFORMLOCATIONPROC)GO_GL_GetProcAddress(
          "glGetUniformLocation");
  GL_Uniform1iFunc = (QS_PFNGLUNIFORM1IPROC)GO_GL_GetProcAddress("glUniform1i");
  GL_Uniform1fFunc = (QS_PFNGLUNIFORM1FPROC)GO_GL_GetProcAddress("glUniform1f");
  GL_Uniform3fFunc = (QS_PFNGLUNIFORM3FPROC)GO_GL_GetProcAddress("glUniform3f");
  GL_Uniform4fFunc = (QS_PFNGLUNIFORM4FPROC)GO_GL_GetProcAddress("glUniform4f");

  if (GL_CreateShaderFunc && GL_DeleteShaderFunc && GL_DeleteProgramFunc &&
      GL_ShaderSourceFunc && GL_CompileShaderFunc && GL_GetShaderivFunc &&
      GL_GetShaderInfoLogFunc && GL_GetProgramivFunc &&
      GL_GetProgramInfoLogFunc && GL_CreateProgramFunc && GL_AttachShaderFunc &&
      GL_LinkProgramFunc && GL_BindAttribLocationFunc && GL_UseProgramFunc &&
      GL_GetAttribLocationFunc && GL_VertexAttribPointerFunc &&
      GL_EnableVertexAttribArrayFunc && GL_DisableVertexAttribArrayFunc &&
      GL_GetUniformLocationFunc && GL_Uniform1iFunc && GL_Uniform1fFunc &&
      GL_Uniform3fFunc && GL_Uniform4fFunc) {
    Con_Printf("FOUND: GLSL\n");
  } else {
    Go_Error("GLSL not available\n");
  }
}

/*
===============
GL_SetupState -- johnfitz

does all the stuff from GL_Init that needs to be done every time a new GL render
context is created
===============
*/
static void GL_SetupState(void) {
  glClearColor(0.15, 0.15, 0.15, 0);
  glCullFace(GL_BACK);
  glFrontFace(GL_CW);
  glEnable(GL_TEXTURE_2D);
  glEnable(GL_ALPHA_TEST);
  glAlphaFunc(GL_GREATER, 0.666);
  glPolygonMode(GL_FRONT_AND_BACK, GL_FILL);
  glShadeModel(GL_FLAT);
  glHint(GL_PERSPECTIVE_CORRECTION_HINT, GL_NICEST);
  glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
  glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_REPEAT);
  glTexParameterf(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_REPEAT);
  glDepthRange(0, 1);
  glDepthFunc(GL_LEQUAL);
}

/*
===============
GL_Init
===============
*/
static void GL_Init(void) {
  int gl_version_major;
  int gl_version_minor;
  const char *gl_vendor = (const char *)glGetString(GL_VENDOR);
  const char *gl_renderer = (const char *)glGetString(GL_RENDERER);
  const char *gl_version = (const char *)glGetString(GL_VERSION);
  const char *gl_extensions = (const char *)glGetString(GL_EXTENSIONS);

  Con_SafePrintf("GL_VENDOR: %s\n", gl_vendor);
  Con_SafePrintf("GL_RENDERER: %s\n", gl_renderer);
  Con_SafePrintf("GL_VERSION: %s\n", gl_version);

  if (gl_version == NULL ||
      sscanf(gl_version, "%d.%d", &gl_version_major, &gl_version_minor) < 2) {
    gl_version_major = 0;
  }
  if (gl_version_major < 2) {
    Go_Error("Need OpenGL version 2 or later");
  }

  GL_CheckExtensions(gl_extensions);

  // johnfitz -- intel video workarounds from Baker
  if (!strcmp(gl_vendor, "Intel")) {
    Con_Printf("Intel Display Adapter detected, enabling gl_clear\n");
    Cbuf_AddText("gl_clear 1");
  }

  GLAlias_CreateShaders();
  GL_ClearBufferBindings();
}

void VID_Init(void) {
  int p, width, height, bpp, display_width, display_height, display_bpp;
  qboolean fullscreen;
  Cvar_FakeRegister(&vid_fullscreen, "vid_fullscreen");
  Cvar_FakeRegister(&vid_width, "vid_width");
  Cvar_FakeRegister(&vid_height, "vid_height");
  Cvar_FakeRegister(&vid_bpp, "vid_bpp");
  Cvar_FakeRegister(&vid_vsync, "vid_vsync");
  Cvar_FakeRegister(&vid_desktopfullscreen, "vid_desktopfullscreen");
  Cvar_FakeRegister(&vid_borderless, "vid_borderless");

  Cmd_AddCommand("vid_restart", VID_Restart);
  Cmd_AddCommand("vid_test", VID_Test);

  if (SDL_InitSubSystem(SDL_INIT_VIDEO) < 0)
    Go_Error_S("Couldn't init SDL video: %v", SDL_GetError());

  {
    SDL_DisplayMode mode;
    if (SDL_GetDesktopDisplayMode(0, &mode) != 0)
      Go_Error("Could not get desktop display mode");

    display_width = mode.w;
    display_height = mode.h;
    display_bpp = SDL_BITSPERPIXEL(mode.format);
  }

  Cvar_SetValueQuick(&vid_bpp, (float)display_bpp);

  // TODO(therjak): It would be good to have read the configs already
  // quakespams reads at least config.cfg here for its cvars. But cvars
  // exist in autoexec.cfg and default.cfg as well.

  VID_InitModelist();

  width = (int)Cvar_GetValue(&vid_width);
  height = (int)Cvar_GetValue(&vid_height);
  bpp = (int)Cvar_GetValue(&vid_bpp);
  fullscreen = (int)Cvar_GetValue(&vid_fullscreen);

  if (CMLCurrent()) {
    width = display_width;
    height = display_height;
    bpp = display_bpp;
    fullscreen = true;
  } else {
    p = CMLWidth();
    if (p >= 0) {
      width = p;
      if (CMLHeight() < 0) height = width * 3 / 4;
    }
    p = CMLHeight();
    if (p >= 0) {
      height = p;
      if (CMLWidth() < 0) width = height * 4 / 3;
    }
    p = CMLBpp();
    if (p >= 0) bpp = p;

    if (CMLWindow())
      fullscreen = false;
    else if (CMLFullscreen())
      fullscreen = true;
  }

  if (!VID_ValidMode(width, height, bpp, fullscreen)) {
    width = (int)Cvar_GetValue(&vid_width);
    height = (int)Cvar_GetValue(&vid_height);
    bpp = (int)Cvar_GetValue(&vid_bpp);
    fullscreen = (int)Cvar_GetValue(&vid_fullscreen);
  }

  if (!VID_ValidMode(width, height, bpp, fullscreen)) {
    width = 640;
    height = 480;
    bpp = display_bpp;
    fullscreen = false;
  }

  SetVID_Initialized(true);

  PL_SetWindowIcon();

  VID_SetMode(width, height, bpp, fullscreen);

  GL_Init();
  GL_SetupState();

  Cvar_FakeRegister(&vid_gamma, "gamma");
  Cvar_FakeRegister(&vid_contrast, "contrast");

  // QuakeSpasm: current vid settings should override config file settings.
  // so we have to lock the vid mode from now until after all config files are
  // read.
  SetVID_Locked(true);

  IN_Init();
}
