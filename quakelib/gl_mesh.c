// gl_mesh.c: triangle model functions

#include "quakedef.h"

/*
=================================================================

ALIAS MODEL DISPLAY LIST GENERATION

=================================================================
*/

qmodel_t *aliasmodel;
aliashdr_t *paliashdr;

int used[8192];  // qboolean

// the command list holds counts and s/t values that are valid for
// every frame
int commands[8192];
int numcommands;

// all frames will have their vertexes rearranged and expanded
// so they are in the order expected by the command list
int vertexorder[8192];
int numorder;

int allverts, alltris;

int stripverts[128];
int striptris[128];
int stripcount;

/*
================
StripLength
================
*/
int StripLength(int starttri, int startv, aliashdr_t *ahdr) {
  int m1, m2;
  int j;
  mtriangle_t *last, *check;
  int k;

  used[starttri] = 2;

  last = &triangles[starttri];

  stripverts[0] = last->vertindex[(startv) % 3];
  stripverts[1] = last->vertindex[(startv + 1) % 3];
  stripverts[2] = last->vertindex[(startv + 2) % 3];

  striptris[0] = starttri;
  stripcount = 1;

  m1 = last->vertindex[(startv + 2) % 3];
  m2 = last->vertindex[(startv + 1) % 3];

// look for a matching triangle
nexttri:
  for (j = starttri + 1, check = &triangles[starttri + 1]; j < ahdr->numtris;
       j++, check++) {
    if (check->facesfront != last->facesfront) continue;
    for (k = 0; k < 3; k++) {
      if (check->vertindex[k] != m1) continue;
      if (check->vertindex[(k + 1) % 3] != m2) continue;

      // this is the next part of the fan

      // if we can't use this triangle, this tristrip is done
      if (used[j]) goto done;

      // the new edge
      if (stripcount & 1)
        m2 = check->vertindex[(k + 2) % 3];
      else
        m1 = check->vertindex[(k + 2) % 3];

      stripverts[stripcount + 2] = check->vertindex[(k + 2) % 3];
      striptris[stripcount] = j;
      stripcount++;

      used[j] = 2;
      goto nexttri;
    }
  }
done:

  // clear the temp used flags
  for (j = starttri + 1; j < ahdr->numtris; j++)
    if (used[j] == 2) used[j] = 0;

  return stripcount;
}

/*
===========
FanLength
===========
*/
int FanLength(int starttri, int startv, aliashdr_t *ahdr) {
  int m1, m2;
  int j;
  mtriangle_t *last, *check;
  int k;

  used[starttri] = 2;

  last = &triangles[starttri];

  stripverts[0] = last->vertindex[(startv) % 3];
  stripverts[1] = last->vertindex[(startv + 1) % 3];
  stripverts[2] = last->vertindex[(startv + 2) % 3];

  striptris[0] = starttri;
  stripcount = 1;

  m1 = last->vertindex[(startv + 0) % 3];
  m2 = last->vertindex[(startv + 2) % 3];

// look for a matching triangle
nexttri:
  for (j = starttri + 1, check = &triangles[starttri + 1]; j < ahdr->numtris;
       j++, check++) {
    if (check->facesfront != last->facesfront) continue;
    for (k = 0; k < 3; k++) {
      if (check->vertindex[k] != m1) continue;
      if (check->vertindex[(k + 1) % 3] != m2) continue;

      // this is the next part of the fan

      // if we can't use this triangle, this tristrip is done
      if (used[j]) goto done;

      // the new edge
      m2 = check->vertindex[(k + 2) % 3];

      stripverts[stripcount + 2] = m2;
      striptris[stripcount] = j;
      stripcount++;

      used[j] = 2;
      goto nexttri;
    }
  }
done:

  // clear the temp used flags
  for (j = starttri + 1; j < ahdr->numtris; j++)
    if (used[j] == 2) used[j] = 0;

  return stripcount;
}

/*
================
BuildTris

Generate a list of trifans or strips
for the model, which holds for all frames
================
*/
void BuildTris(aliashdr_t *ahdr) {
  int i, j, k;
  int startv;
  float s, t;
  int len, bestlen, besttype;
  int bestverts[1024];
  int besttris[1024];
  int type;

  //
  // build tristrips
  //
  numorder = 0;
  numcommands = 0;
  memset(used, 0, sizeof(used));
  for (i = 0; i < ahdr->numtris; i++) {
    // pick an unused triangle and start the trifan
    if (used[i]) continue;

    bestlen = 0;
    besttype = 0;
    for (type = 0; type < 2; type++)
    //	type = 1;
    {
      for (startv = 0; startv < 3; startv++) {
        if (type == 1)
          len = StripLength(i, startv, ahdr);
        else
          len = FanLength(i, startv, ahdr);
        if (len > bestlen) {
          besttype = type;
          bestlen = len;
          for (j = 0; j < bestlen + 2; j++) bestverts[j] = stripverts[j];
          for (j = 0; j < bestlen; j++) besttris[j] = striptris[j];
        }
      }
    }

    // mark the tris on the best strip as used
    for (j = 0; j < bestlen; j++) used[besttris[j]] = 1;

    if (besttype == 1)
      commands[numcommands++] = (bestlen + 2);
    else
      commands[numcommands++] = -(bestlen + 2);

    for (j = 0; j < bestlen + 2; j++) {
      int tmp;

      // emit a vertex into the reorder buffer
      k = bestverts[j];
      vertexorder[numorder++] = k;

      // emit s/t coords into the commands stream
      s = stverts[k].s;
      t = stverts[k].t;
      if (!triangles[besttris[0]].facesfront && stverts[k].onseam)
        s += ahdr->skinwidth / 2;  // on back side
      s = (s + 0.5) / ahdr->skinwidth;
      t = (t + 0.5) / ahdr->skinheight;

      //	*(float *)&commands[numcommands++] = s;
      //	*(float *)&commands[numcommands++] = t;
      // NOTE: 4 == sizeof(int)
      //	   == sizeof(float)
      memcpy(&tmp, &s, 4);
      commands[numcommands++] = tmp;
      memcpy(&tmp, &t, 4);
      commands[numcommands++] = tmp;
    }
  }

  commands[numcommands++] = 0;  // end of list marker

  Con_DPrintf2("%3i tri %3i vert %3i cmd\n", ahdr->numtris, numorder,
               numcommands);

  allverts += numorder;
  alltris += ahdr->numtris;
}

static void GL_MakeAliasModelDisplayLists_VBO(aliashdr_t *ahdr);
static void GLMesh_LoadVertexBuffer(qmodel_t *m, const aliashdr_t *hdr);

/*
================
GL_MakeAliasModelDisplayLists
================
*/
void GL_MakeAliasModelDisplayLists(qmodel_t *m, aliashdr_t *hdr) {
  int i, j;
  int *cmds;
  trivertx_t *verts;
  int count;      // johnfitz -- precompute texcoords for padded skins
  int *loadcmds;  // johnfitz

  aliasmodel = m;
  paliashdr = hdr;  // (aliashdr_t *)Mod_Extradata (m);

  // johnfitz -- generate meshes
  Con_DPrintf2("meshing %s...\n", m->name);
  BuildTris(hdr);

  // save the data out

  paliashdr->poseverts = numorder;

  cmds = (int *)Hunk_Alloc(numcommands * 4);
  paliashdr->commands = (byte *)cmds - (byte *)paliashdr;

  // johnfitz -- precompute texcoords for padded skins
  loadcmds = commands;
  while (1) {
    *cmds++ = count = *loadcmds++;

    if (!count) break;

    if (count < 0) count = -count;

    do {
      *(float *)cmds++ = (*(float *)loadcmds++);
      *(float *)cmds++ = (*(float *)loadcmds++);
    } while (--count);
  }
  // johnfitz

  verts = (trivertx_t *)Hunk_Alloc(paliashdr->numposes * paliashdr->poseverts *
                                   sizeof(trivertx_t));
  paliashdr->posedata = (byte *)verts - (byte *)paliashdr;
  for (i = 0; i < paliashdr->numposes; i++)
    for (j = 0; j < numorder; j++) *verts++ = poseverts[i][vertexorder[j]];

  // ericw
  GL_MakeAliasModelDisplayLists_VBO(hdr);
}

unsigned int r_meshindexbuffer = 0;
unsigned int r_meshvertexbuffer = 0;

/*
================
GL_MakeAliasModelDisplayLists_VBO

Saves data needed to build the VBO for this model on the hunk. Afterwards this
is copied to Mod_Extradata.

Original code by MH from RMQEngine
================
*/
void GL_MakeAliasModelDisplayLists_VBO(aliashdr_t *ahdr) {
  int i, j;
  int maxverts_vbo;
  trivertx_t *verts;
  unsigned short *indexes;
  aliasmesh_t *desc;

  // first, copy the verts onto the hunk
  verts = (trivertx_t *)Hunk_Alloc(paliashdr->numposes * paliashdr->numverts *
                                   sizeof(trivertx_t));
  paliashdr->vertexes = (byte *)verts - (byte *)paliashdr;
  for (i = 0; i < paliashdr->numposes; i++)
    for (j = 0; j < paliashdr->numverts; j++)
      verts[i * paliashdr->numverts + j] = poseverts[i][j];

  // there can never be more than this number of verts and we just put them all
  // on the hunk
  maxverts_vbo = ahdr->numtris * 3;
  desc = (aliasmesh_t *)Hunk_Alloc(sizeof(aliasmesh_t) * maxverts_vbo);

  // there will always be this number of indexes
  indexes = (unsigned short *)Hunk_Alloc(sizeof(unsigned short) * maxverts_vbo);

  ahdr->indexes = (intptr_t)indexes - (intptr_t)ahdr;
  ahdr->meshdesc = (intptr_t)desc - (intptr_t)ahdr;
  ahdr->numindexes = 0;
  ahdr->numverts_vbo = 0;

  for (i = 0; i < ahdr->numtris; i++) {
    for (j = 0; j < 3; j++) {
      int v;

      // index into hdr->vertexes
      unsigned short vertindex = triangles[i].vertindex[j];

      // basic s/t coords
      int s = stverts[vertindex].s;
      int t = stverts[vertindex].t;

      // check for back side and adjust texcoord s
      if (!triangles[i].facesfront && stverts[vertindex].onseam)
        s += ahdr->skinwidth / 2;

      // see does this vert already exist
      for (v = 0; v < ahdr->numverts_vbo; v++) {
        // it could use the same xyz but have different s and t
        if (desc[v].vertindex == vertindex && (int)desc[v].st[0] == s &&
            (int)desc[v].st[1] == t) {
          // exists; emit an index for it
          indexes[ahdr->numindexes++] = v;

          // no need to check any more
          break;
        }
      }

      if (v == ahdr->numverts_vbo) {
        // doesn't exist; emit a new vert and index
        indexes[ahdr->numindexes++] = ahdr->numverts_vbo;

        desc[ahdr->numverts_vbo].vertindex = vertindex;
        desc[ahdr->numverts_vbo].st[0] = s;
        desc[ahdr->numverts_vbo++].st[1] = t;
      }
    }
  }

  // upload immediately
  GLMesh_LoadVertexBuffer(aliasmodel, ahdr);
}

/*
================
GLMesh_LoadVertexBuffer

Upload the given alias model's mesh to a VBO

Original code by MH from RMQEngine
================
*/
static void GLMesh_LoadVertexBuffer(qmodel_t *m, const aliashdr_t *hdr) {
  int totalvbosize = 0;
  const aliasmesh_t *desc;
  const short *indexes;
  const trivertx_t *trivertexes;
  byte *vbodata;
  int f;

  // count the sizes we need

  // ericw -- RMQEngine stored these vbo*ofs values in aliashdr_t, but we must
  // not
  // mutate Mod_Extradata since it might be reloaded from disk, so I moved them
  // to qmodel_t
  // (test case: roman1.bsp from arwop, 64mb heap)
  m->vboindexofs = 0;

  m->vboxyzofs = 0;
  totalvbosize += (hdr->numposes * hdr->numverts_vbo *
                   sizeof(meshxyz_t));  // ericw -- what RMQEngine called
                                        // nummeshframes is called numposes in
                                        // QuakeSpasm

  m->vbostofs = totalvbosize;
  totalvbosize += (hdr->numverts_vbo * sizeof(meshst_t));

  if (!hdr->numindexes) return;
  if (!totalvbosize) return;

  // grab the pointers to data in the extradata

  desc = (aliasmesh_t *)((byte *)hdr + hdr->meshdesc);
  indexes = (short *)((byte *)hdr + hdr->indexes);
  trivertexes = (trivertx_t *)((byte *)hdr + hdr->vertexes);

  // upload indices buffer

  glDeleteBuffers(1, &m->meshindexesvbo);
  glGenBuffers(1, &m->meshindexesvbo);
  glBindBuffer(GL_ELEMENT_ARRAY_BUFFER, m->meshindexesvbo);
  glBufferData(GL_ELEMENT_ARRAY_BUFFER,
               hdr->numindexes * sizeof(unsigned short), indexes,
               GL_STATIC_DRAW);

  // create the vertex buffer (empty)

  vbodata = (byte *)malloc(totalvbosize);
  memset(vbodata, 0, totalvbosize);

  // fill in the vertices at the start of the buffer
  for (f = 0; f < hdr->numposes; f++)  // ericw -- what RMQEngine called
                                       // nummeshframes is called numposes in
                                       // QuakeSpasm
  {
    int v;
    meshxyz_t *xyz =
        (meshxyz_t *)(vbodata + (f * hdr->numverts_vbo * sizeof(meshxyz_t)));
    const trivertx_t *tv = trivertexes + (hdr->numverts * f);

    for (v = 0; v < hdr->numverts_vbo; v++) {
      trivertx_t trivert = tv[desc[v].vertindex];

      xyz[v].xyz[0] = trivert.v[0];
      xyz[v].xyz[1] = trivert.v[1];
      xyz[v].xyz[2] = trivert.v[2];
      xyz[v].xyz[3] = 1;  // need w 1 for 4 byte vertex compression

      // map the normal coordinates in [-1..1] to [-127..127] and store in an
      // unsigned char.
      // this introduces some error (less than 0.004), but the normals were very
      // coarse
      // to begin with
      xyz[v].normal[0] = 127 * R_avertexnormals(trivert.lightnormalindex, 0);
      xyz[v].normal[1] = 127 * R_avertexnormals(trivert.lightnormalindex, 1);
      xyz[v].normal[2] = 127 * R_avertexnormals(trivert.lightnormalindex, 2);
      xyz[v].normal[3] = 0;  // unused; for 4-byte alignment
    }
  }

  // fill in the ST coords at the end of the buffer
  {
    meshst_t *st;
    st = (meshst_t *)(vbodata + m->vbostofs);
    for (f = 0; f < hdr->numverts_vbo; f++) {
      st[f].st[0] = ((float)desc[f].st[0] + 0.5f) / (float)hdr->skinwidth;
      st[f].st[1] = ((float)desc[f].st[1] + 0.5f) / (float)hdr->skinheight;
    }
  }

  // upload vertexes buffer
  glDeleteBuffers(1, &m->meshvbo);
  glGenBuffers(1, &m->meshvbo);
  glBindBuffer(GL_ARRAY_BUFFER, m->meshvbo);
  glBufferData(GL_ARRAY_BUFFER, totalvbosize, vbodata, GL_STATIC_DRAW);

  free(vbodata);

  // invalidate the cached bindings
  GL_ClearBufferBindings();
}

/*
================
GLMesh_LoadVertexBuffers

Loop over all precached alias models, and upload each one to a VBO.
================
*/
void GLMesh_LoadVertexBuffers(void) {
  int j;
  qmodel_t *m;
  const aliashdr_t *hdr;

  for (j = 1; j < MAX_MODELS; j++) {
    if (!(m = cl.model_precache[j])) break;
    if (m->Type != mod_alias) continue;

    hdr = (const aliashdr_t *)Mod_Extradata(m);

    GLMesh_LoadVertexBuffer(m, hdr);
  }
}

/*
================
GLMesh_DeleteVertexBuffers

Delete VBOs for all loaded alias models
================
*/
void GLMesh_DeleteVertexBuffers(void) {
  int j;
  qmodel_t *m;

  for (j = 1; j < MAX_MODELS; j++) {
    if (!(m = cl.model_precache[j])) break;
    if (m->Type != mod_alias) continue;

    glDeleteBuffers(1, &m->meshvbo);
    m->meshvbo = 0;

    glDeleteBuffers(1, &m->meshindexesvbo);
    m->meshindexesvbo = 0;
  }

  GL_ClearBufferBindings();
}
