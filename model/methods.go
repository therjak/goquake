package model

import (
	"bytes"
	"fmt"
	"log"
	"quake/math/vec"
)

func (m *QModel) PointInLeaf(p vec.Vec3) (*MLeaf, error) {
	if m == nil || len(m.Nodes) == 0 {
		return nil, fmt.Errorf("Mod_PointInLeaf: bad model")
	}

	node := m.Node
	for {
		if node.Contents() < 0 {
			return node.(*MLeaf), nil
		}
		n := node.(*MNode)
		plane := n.Plane
		d := vec.Dot(p, plane.Normal) - plane.Dist
		if d > 0 {
			node = n.Children[0]
		} else {
			node = n.Children[1]
		}
	}
	return nil, nil
}

func (m *QModel) DecompressVis(in []byte) []byte {
	row := (len(m.Leafs) + 7) >> 3

	if len(in) == 0 {
		// no vis info, so make all visible
		for i := 0; i < row; i++ {
			decompressedVis[i] = 0xff
		}
		return decompressedVis[:row]
	}

	// 'in' is compressed and looks like
	// 70550311
	// and gets uncompressed to
	// 700000500011	(7 5x0 5 3x0 1 1)

	j := 0
	for i := 0; i < len(in); i++ {
		if in[i] != 0 {
			decompressedVis[j] = in[i]
			j++
		} else {
			i++
			if i >= len(in) {
				log.Printf("Faulty vis data in model %s", m.Name)
				break
			}
			for c := in[i]; c > 0; c-- {
				decompressedVis[j] = 0
				j++
			}
			if j >= row {
				break
			}
		}
	}
	return decompressedVis[:row] // should this be :j?
}

var (
	noVis           []byte
	decompressedVis []byte
	fatpvs          []byte
)

func init() {
	noVis = bytes.Repeat([]byte{0xff}, MAX_MAP_LEAFS/8)
	decompressedVis = make([]byte, MAX_MAP_LEAFS/8)
	fatpvs = make([]byte, MAX_MAP_LEAFS/8)
}

func (m *QModel) LeafPVS(leaf *MLeaf) []byte {
	if leaf == m.Leafs[0] { // What should this actually do?
		return noVis
	}
	return m.DecompressVis(leaf.CompressedVis)
}

/*
The PVS must include a small area around the client to allow head bobbing
or other small motion on the client side.  Otherwise, a bob might cause an
entity that should be visible to not show up, especially when the bob
crosses a waterline.
*/
func (m *QModel) addToFatPVS(org vec.Vec3, n Node, fpvs *[]byte) {
	node := n
	for {
		if node.Contents() < 0 {
			// if this is a leaf, accumulate the pvs bits
			if node.Contents() != CONTENTS_SOLID {
				pvs := m.LeafPVS(node.(*MLeaf))
				for i := range *fpvs {
					(*fpvs)[i] = pvs[i]
				}
			}
			return
		}
		no := node.(*MNode)
		plane := no.Plane
		d := vec.Dot(org, plane.Normal) - plane.Dist
		if d > 8 {
			node = no.Children[0]
		} else if d < -8 {
			node = no.Children[1]
		} else { // go down both
			m.addToFatPVS(org, no.Children[0], fpvs)
			node = no.Children[1]
		}
	}
}

//Calculates a PVS that is the inclusive or of all leafs within 8 pixels of the
//given point.
func (m *QModel) FatPVS(org vec.Vec3) []byte {
	fatbytes := ((len(m.Leafs) + 7) >> 3)
	pvs := fatpvs[:fatbytes]
	for i := range pvs {
		pvs[i] = 0
	}
	m.addToFatPVS(org, m.Node, &pvs)
	return pvs
}
