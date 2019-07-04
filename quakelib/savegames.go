package quakelib

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"path/filepath"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
	"quake/execute"
	"quake/protos"
	"strings"
)

func init() {
	cmd.AddCommand("save", saveGame)
	cmd.AddCommand("load", loadGame)
}

func saveGameComment() string {
	ln := sv.worldModel.Name // cl.levelname
	km := cl.stats.monsters
	tm := cl.stats.totalMonsters
	// somehow nobody can count, we should have 39 chars total available, why clip at 22 for the map?
	return fmt.Sprintf("%-22s kills:%3d/%3d", ln, km, tm)
}

func saveGame(args []cmd.QArg, _ int) {
	if !execute.IsSrcCommand() {
		return
	}
	if !sv.active {
		conlog.Printf("Not playing a local game.\n")
		return
	}

	if cl.intermission != 0 {
		conlog.Printf("Can't save in intermission.\n")
		return
	}

	if svs.maxClients != 1 {
		conlog.Printf("Can't save multiplayer games.\n")
		return
	}

	if len(args) != 1 {
		conlog.Printf("save <savename> : save a game\n")
		return
	}

	filename := args[0].String()
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		// We will add filename to the gamedir so we with this we are always inside the gamedir
		conlog.Printf("Relative pathnames are not allowed.\n")
		return
	}

	if EntVars(sv_clients[0].edictId).Health <= 0 {
		conlog.Printf("Can't savegame with a dead player\n")
		return
	}

	fullname := filepath.Join(GameDirectory(), filename)
	if filepath.Ext(fullname) != ".sav" {
		fullname = fullname + ".sav"
	}

	conlog.Printf("Saving game to %s...\n", fullname)

	data := &protos.SaveGame{
		Comment: saveGameComment(),
		// SpawnParms: []float32
		CurrentSkill: int32(cvars.Skill.Value()),
		MapName:      sv.name,
		MapTime:      sv.time,
		//LightStyles: []string
		//Globals: protos.Globals
		//Edicts: []protos.Edict
	}

	/*
	   for (i = 0; i < NUM_SPAWN_PARMS; i++)
	     fprintf(f, "%f\n", GetClientSpawnParam(0, i));

	   for (i = 0; i < MAX_LIGHTSTYLES; i++) {
	     if (sv.lightstyles[i])
	       fprintf(f, "%s\n", sv.lightstyles[i]);
	     else
	       fprintf(f, "m\n");
	   }

	   ED_WriteGlobals(f);
	   for (i = 0; i < SV_NumEdicts(); i++) {
	     ED_Write(f, i);
	   }
	*/
	out, err := proto.Marshal(data)
	if err != nil {
		conlog.Printf("failed to encode savegame.\n")
		return
	}
	if err := ioutil.WriteFile(fullname, out, 0660); err != nil {
		conlog.Printf("ERROR: couldn't write file.\n")
		return
	}
	conlog.Printf("done.\n")
}

func loadGame(args []cmd.QArg, _ int) {
}

/*
void Host_Loadgame_f(void) {
  char name[MAX_OSPATH];
  FILE *f;
  char mapname[MAX_QPATH];
  float time, tfloat;
  char str[32768];
  const char *start;
  int i, r;
  int entnum;
  int version;
  float spawn_parms[NUM_SPAWN_PARMS];

  if (!IsSrcCommand()) return;

  if (Cmd_Argc() != 2) {
    Con_Printf("load <savename> : load a game\n");
    return;
  }

  CLS_StopDemoCycle();  // stop demo loop in case this fails

  q_snprintf(name, sizeof(name), "%s/%s", Com_Gamedir(), Cmd_Argv(1));
  COM_AddExtension(name, ".sav", sizeof(name));

  // we can't call SCR_BeginLoadingPlaque, because too much stack space has
  // been used.  The menu calls it before stuffing loadgame command
  //	SCR_BeginLoadingPlaque ();

  Con_Printf("Loading game from %s...\n", name);
  f = fopen(name, "r");
  if (!f) {
    Con_Printf("ERROR: couldn't open.\n");
    return;
  }

  fscanf(f, "%i\n", &version);
  if (version != SAVEGAME_VERSION) {
    fclose(f);
    Con_Printf("Savegame is version %i, not %i\n", version, SAVEGAME_VERSION);
    return;
  }
  fscanf(f, "%s\n", str);
  for (i = 0; i < NUM_SPAWN_PARMS; i++) fscanf(f, "%f\n", &spawn_parms[i]);
  // this silliness is so we can load 1.06 save files, which have float skill
  // values
  fscanf(f, "%f\n", &tfloat);
  current_skill = (int)(tfloat + 0.1);
  Cvar_SetValue("skill", (float)current_skill);

  fscanf(f, "%s\n", mapname);
  fscanf(f, "%f\n", &time);

  CL_Disconnect_f();

  SV_SpawnServer(mapname);

  if (!SV_Active()) {
    fclose(f);
    Con_Printf("Couldn't load map\n");
    return;
  }
  SV_SetPaused(true);  // pause until all clients connect
  SV_SetLoadGame(true);

  // load the light styles

  for (i = 0; i < MAX_LIGHTSTYLES; i++) {
    fscanf(f, "%s\n", str);
    sv.lightstyles[i] = (const char *)Hunk_Strdup(str, "lightstyles");
    SetSVLightStyles(i, sv.lightstyles[i]);
  }

  // load the edicts out of the savegame file
  entnum = -1;  // -1 is the globals
  while (!feof(f)) {
    qboolean inside_string = false;
    for (i = 0; i < (int)sizeof(str) - 1; i++) {
      r = fgetc(f);
      if (r == EOF || !r) break;
      str[i] = r;
      if (r == '"') {
        inside_string = !inside_string;
      } else if (r == '}' && !inside_string)  // only handle } characters
                                              // outside of quoted strings
      {
        i++;
        break;
      }
    }
    if (i == (int)sizeof(str) - 1) {
      fclose(f);
      Go_Error("Loadgame buffer overflow");
    }
    str[i] = 0;
    start = str;
    start = COM_Parse(str);
    if (!com_token[0]) break;  // end of file
    if (strcmp(com_token, "{")) {
      fclose(f);
      Go_Error("First token isn't a brace");
    }

    if (entnum == -1) {  // parse the global vars
      ED_ParseGlobals(start);
    } else {  // parse an edict
      if (entnum < SV_NumEdicts()) {
        EDICT_SETFREE(entnum, false);
        TTClearEntVars(entnum);
      } else {
        TT_ClearEdict(entnum);
      }
      ED_ParseEdict(start, entnum);

      // link it into the bsp tree
      if (!EDICT_FREE(entnum)) SV_LinkEdict(entnum, false);
    }

    entnum++;
  }

  SV_SetNumEdicts(entnum);
  SV_SetTime(time);

  fclose(f);

  for (i = 0; i < NUM_SPAWN_PARMS; i++)
    SetClientSpawnParam(0, i, spawn_parms[i]);

  if (CLS_GetState() != ca_dedicated) {
    CL_EstablishConnection("local");
    Host_Reconnect_f();
  }
}
*/
