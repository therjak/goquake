// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

/*
	func TraceLine(start, end, impact *C.float) {
		s := p2v3(start)
		e := p2v3(end)
		trace := trace{}
		recursiveHullCheck(&cl.worldModel.Hulls[0], 0, 0, 1, s, e, &trace)
		*C.cfp(0, impact) = C.float(trace.EndPos[0])
		*C.cfp(1, impact) = C.float(trace.EndPos[1])
		*C.cfp(2, impact) = C.float(trace.EndPos[2])
	}
*/
func Chase_UpdateForDrawing() {
	// orient camera based on client. called before drawing
	/*
	  int i;
	  vec3_t forward, up, right;
	  vec3_t ideal, crosshair, temp;
	  vec3_t clviewangles;
	  clviewangles[PITCH] = cl.pitch
	  clviewangles[YAW] = cl.yaw
	  clviewangles[ROLL] = cl.roll

	  AngleVectors(clviewangles, forward, right, up);

	  // calc ideal camera location before checking for walls
	  for (i = 0; i < 3; i++)
	    ideal[i] = cl.WeaponEntity().Origin[i] - forward[i] * Cvar_GetValue(&chase_back) +
	               right[i] * Cvar_GetValue(&chase_right);
	  //+ up[i]*Cvar_GetValue(&chase_up);
	  ideal[2] = cl.WeaponEntity().Origin[2] + Cvar_GetValue(&chase_up);

	  // make sure camera is not in or behind a wall
	  TraceLine(r_refdef.vieworg, ideal, temp);
	  if (VectorLength(temp) != 0) VectorCopy(temp, ideal);

	  // place camera
	  VectorCopy(ideal, r_refdef.vieworg);

	  // find the spot the player is looking at
	  VectorMA(cl.WeaponEntity().Origin, 4096, forward, temp);
	  TraceLine(cl.WeaponEntity().Origin, temp, crosshair);

	  // calculate camera angles to look at the same spot
	  VectorSubtract(crosshair, r_refdef.vieworg, temp);
	  VectorAngles(temp, r_refdef.viewangles);
	  if (r_refdef.viewangles[PITCH] == 90 || r_refdef.viewangles[PITCH] == -90)
	    r_refdef.viewangles[YAW] = cl.yaw
	*/
}
