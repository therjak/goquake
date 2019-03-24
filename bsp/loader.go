package bsp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"quake/filesystem"
	qm "quake/model"
)

var (
	polyMagic   = [4]byte{'I', 'D', 'P', 'O'}
	spriteMagic = [4]byte{'I', 'D', 'S', 'P'}
)

func LoadModel(name string) (*qm.QModel, error) {
	// TODO: Add cache

	b, err := filesystem.GetFileContents(name)
	if err != nil {
		return nil, err
	}
	var magic [4]byte
	copy(magic[:], b)
	switch magic {
	case polyMagic:
		log.Printf("Got a poly %s", name)
	// LoadAliasModel, this is a .mdl
	case spriteMagic:
		log.Printf("Got a sprite %s", name)
		// LoadSpriteModel, this is a .spr
	default:
		log.Printf("Got a bsp %s", name)
		return LoadBSP(name, b)
	}
	return nil, nil
}

const (
	bspVersion       = 29
	bsp2Version_2psb = 'B'<<24 | 'S'<<16 | 'P'<<8 | '2'
	bsp2Version_bsp2 = '2'<<24 | 'P'<<16 | 'S'<<8 | 'B'
)

func LoadBSP(name string, data []byte) (*qm.QModel, error) {
	ret := &qm.QModel{
		Name: name,
		Type: qm.ModBrush,
		// Mins, Maxs, ClipMins, Clipmaxs Vec3
		// NumSubmodels int
		// leafs []*qm.MLeaf
		// Node qm.Node
		// Hulls [4]qm.Hull
	}
	buf := bytes.NewReader(data)
	h := header{}
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	fs := func(d directory, data []byte) []byte {
		return data[d.Offset : d.Offset+d.Size]
	}
	switch h.Version {
	case bspVersion:
		log.Printf("Got V0 bsp: %v", h)
		// loadVertexes(fs(h.Vertexes, data),ret)
		// loadEdgesV0(fs(h.Edges, data),ret)
		// loadSurfaceEdges(fs(h.SurfaceEdges, data),ret)
		// loadTextures(fs(h.Textures, data),ret)
		// loadLighting(fs(h.Lighting, data),ret)
		// loadPlanes(fs(h.Planes, data),ret)
		// loadTextinfo(fs(h.Texinfo , data),ret)
		sfaces, err := loadFacesV0(fs(h.Faces, data))
		if err != nil {
			return nil, err
		}
		msurfaces, err := buildSurfacesV0(sfaces, ret.Planes, ret.Texinfos)
		if err != nil {
			return nil, err
		}
		ret.Surfaces = msurfaces
		msurf, err := loadMarkSurfacesV0(fs(h.MarkSurfaces, data))
		if err != nil {
			return nil, err
		}
		mms, err := buildMarkSurfacesV0(msurf, ret.Surfaces)
		if err != nil {
			return nil, err
		}
		ret.MarkSurfaces = mms
		ret.VisData = fs(h.Visibility, data)
		leafs, err := loadLeafsV0(fs(h.Leafs, data))
		if err != nil {
			return nil, err
		}
		ml, err := buildLeafsV0(leafs, ret.MarkSurfaces, ret.VisData)
		if err != nil {
			return nil, err
		}
		ret.Leafs = ml
		nodes, err := loadNodesV0(fs(h.Nodes, data))
		if err != nil {
			return nil, err
		}
		mn, err := buildNodesV0(nodes, ret.Leafs, ret.Planes)
		if err != nil {
			return nil, err
		}
		ret.Nodes = mn

		// loadClipNodesV0(fs(h.ClipNodes , data),ret)
		// loadEntities(fs(h.Entities , data),ret)
		// loadModels(fs(h.Models , data),ret)
		// makeHull0(ret)

		// read leafs
		// read nodes
		// read clipnodes
		// read 'submodels', submodel[0] is the 'map'
		// HeadNode [0] == first bsp node index
		// [1] == first clip node index
		// [2] == last clip node index
		// [3] usually 0

	case bsp2Version_2psb:
		log.Printf("Got V1 bsp: %v", h)
	case bsp2Version_bsp2:
		log.Printf("Got V2 bsp: %v", h)
	default:
		log.Printf("Version %v", h.Version)
	}

	return ret, nil
}

func buildMarkSurfacesV0(marks []int, surfaces []*qm.Surface) ([]*qm.Surface, error) {
	ret := make([]*qm.Surface, len(marks))
	for _, m := range marks {
		if m >= len(surfaces) {
			return nil, fmt.Errorf("MarkSurfaces out of bounds")
		}
		ret = append(ret, surfaces[m])
	}
	return ret, nil
}

func loadFacesV0(date []byte) ([]*faceV0, error) {
	// TODO
	return nil, nil
}

func buildSurfacesV0(f []*faceV0, plane []*qm.Plane, texinfo []*qm.Texinfo) ([]*qm.Surface, error) {
	// TODO
	return nil, nil
}

func loadMarkSurfacesV0(data []byte) ([]int, error) {
	ret := []int{}
	buf := bytes.NewReader(data)
	for {
		var val int16
		err := binary.Read(buf, binary.LittleEndian, &val)
		switch err {
		default:
			return nil, err
		case io.EOF:
			return ret, nil
		case nil:
			ret = append(ret, int(val))
		}
	}
}

func buildLeafsV0(ls []*leafV0, ms []*qm.Surface, vd []byte) ([]*qm.MLeaf, error) {
	ret := make([]*qm.MLeaf, len(ls))
	for _, l := range ls {
		nl := &qm.MLeaf{
			NodeBase: qm.NewNodeBase(int(l.Type), 0, [6]float32{
				float32(l.Box[0]), float32(l.Box[1]), float32(l.Box[2]),
				float32(l.Box[3]), float32(l.Box[4]), float32(l.Box[5])}),
			CompressedVis:     vd[l.VisOfs:],
			MarkSurfaces:      ms[l.FirstMarkSurface : l.FirstMarkSurface+l.MarkSurfaceCount],
			AmbientSoundLevel: [4]byte{l.Ambients[0], l.Ambients[1], l.Ambients[2], l.Ambients[3]},
		}
		ret = append(ret, nl)
	}
	return ret, nil
}

func loadLeafsV0(data []byte) ([]*leafV0, error) {
	ret := []*leafV0{}
	buf := bytes.NewReader(data)
	for {
		l := &leafV0{}
		err := binary.Read(buf, binary.LittleEndian, l)
		switch err {
		default:
			return nil, err
		case io.EOF:
			return ret, nil
		case nil:
			ret = append(ret, l)
		}
	}
}

func buildNodesV0(nd []*nodeV0, leafs []*qm.MLeaf, planes []*qm.Plane) ([]*qm.MNode, error) {
	ret := make([]*qm.MNode, len(nd))
	for _, n := range nd {
		nn := &qm.MNode{
			NodeBase: qm.NewNodeBase(0, 0, [6]float32{
				float32(n.Box[0]), float32(n.Box[1]), float32(n.Box[2]),
				float32(n.Box[3]), float32(n.Box[4]), float32(n.Box[5])}),
			// Children:  delay untill we got all nodes
			Plane:        planes[int(n.PlaneID)],
			FirstSurface: uint32(n.FirstSurface),
			SurfaceCount: uint32(n.SurfaceCount),
		}
		ret = append(ret, nn)
	}
	getChild := func(id int) qm.Node {
		if id < len(ret) {
			return ret[id]
		}
		p := 65535 - id // this is intentionally, -1 is leaf 0
		if p < len(leafs) {
			return leafs[p]
		}
		log.Printf("No Child. Got child id %d of %d, p %d of %d. ",
			id, len(ret), p, len(leafs))
		return nil
	}
	for i, n := range nd {
		ret[i].Children[0] = getChild(int(n.Children[0]))
		ret[i].Children[1] = getChild(int(n.Children[1]))
	}
	return ret, nil
}

func loadNodesV0(data []byte) ([]*nodeV0, error) {
	ret := []*nodeV0{}
	buf := bytes.NewReader(data)
	for {
		n := &nodeV0{}
		err := binary.Read(buf, binary.LittleEndian, n)
		switch err {
		default:
			return nil, err
		case io.EOF:
			return ret, nil
		case nil:
			ret = append(ret, n)
		}
	}
}

// func loadClipNodes(fs(h.ClipNodes , data),ret)
// func loadEntities(fs(h.Entities , data),ret)
// func loadModes(fs(h.Models , data),ret)
// func makeHull0(ret)
