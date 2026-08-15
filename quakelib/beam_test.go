// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"testing"

	"goquake/math/vec"
)

type mockModel struct{}

func (m *mockModel) Name() string   { return "test.mdl" }
func (m *mockModel) Mins() vec.Vec3 { return vec.Vec3{} }
func (m *mockModel) Maxs() vec.Vec3 { return vec.Vec3{} }
func (m *mockModel) Flags() int     { return 0 }

func TestUpdateTempEntities_BeamSegmentsAndRotation(t *testing.T) {
	clearBeams()
	ClearVisibleEntities()
	clientTempEntities = make([]Entity, 0, 256)

	c := &Client{
		time:       1.0,
		viewentity: 1,
	}

	mock := &mockModel{}
	beams[0] = beam{
		model:   mock,
		endTime: 2.0,
		start:   vec.Vec3{0, 0, 0},
		end:     vec.Vec3{90, 0, 0},
		entity:  2,
	}

	c.updateTempEntities()

	if len(clientTempEntities) != 3 {
		t.Fatalf("expected 3 beam segments for distance 90, got %d", len(clientTempEntities))
	}

	expectedOrigins := []vec.Vec3{
		{0, 0, 0},
		{30, 0, 0},
		{60, 0, 0},
	}

	for i, expected := range expectedOrigins {
		ent := clientTempEntities[i]
		if ent.Origin != expected {
			t.Errorf("segment %d origin = %v, expected %v", i, ent.Origin, expected)
		}
		// Roll angle should be in [0, 360)
		roll := ent.Angles[2]
		if roll < 0 || roll >= 360 {
			t.Errorf("segment %d roll angle = %v, expected in [0, 360)", i, roll)
		}
	}

	// Verify roll rotation varies across time
	c.time = 2.5
	beams[0].endTime = 3.0
	c.updateTempEntities()

	if len(clientTempEntities) != 3 {
		t.Fatalf("expected 3 beam segments, got %d", len(clientTempEntities))
	}
	for i := range clientTempEntities {
		roll := clientTempEntities[i].Angles[2]
		if roll < 0 || roll >= 360 {
			t.Errorf("segment %d roll angle = %v, expected in [0, 360)", i, roll)
		}
	}
}
