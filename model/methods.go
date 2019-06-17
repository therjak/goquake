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

	node := Node(m.Nodes[0])
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
	row := (len(m.Leafs) + 7) / 8

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
	if j > row {
		log.Printf("Strange vis data in model %s", m.Name)
	}
	return decompressedVis[:row]
}

var (
	noVis           []byte
	decompressedVis []byte
)

func init() {
	noVis = bytes.Repeat([]byte{0xff}, MAX_MAP_LEAFS/8)
	decompressedVis = make([]byte, MAX_MAP_LEAFS/8)
}

func (m *QModel) LeafPVS(leaf *MLeaf) []byte {
	// if (leaf == model->leafs) { // What should this actually do?
	//	return noVis
	//}
	return m.DecompressVis(leaf.CompressedVis)
}
