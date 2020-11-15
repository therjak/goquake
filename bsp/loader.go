package bsp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/therjak/goquake/math/vec"
	qm "github.com/therjak/goquake/model"
)

func init() {
	qm.Register(bspVersion, Load)
	qm.Register(bsp2Version2psb, Load)
	qm.Register(bsp2Versionbsp2, Load)
}

const (
	bspVersion      = 29
	bsp2Version2psb = 'B'<<24 | 'S'<<16 | 'P'<<8 | '2'
	bsp2Versionbsp2 = '2'<<24 | 'P'<<16 | 'S'<<8 | 'B'
)

func Load(name string, data []byte) ([]*qm.QModel, error) {
	var ret []*qm.QModel
	mod := &qm.QModel{
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
		vertexes, err := loadVertexes(fs(h.Vertexes, data))
		if err != nil {
			return nil, err
		}
		mod.Vertexes = vertexes
		edges, err := loadEdgesV0(fs(h.Edges, data))
		if err != nil {
			return nil, err
		}
		mod.Edges = edges
		// TODO: loadSurfaceEdges(fs(h.SurfaceEdges, data),ret)
		textures, err := loadTextures(fs(h.Textures, data))
		if err != nil {
			return nil, err
		}
		mod.Textures = textures
		// TODO: loadLighting(fs(h.Lighting, data),ret)
		splanes, err := loadPlanes(fs(h.Planes, data))
		if err != nil {
			return nil, err
		}
		mod.Planes = buildPlanes(splanes)
		texInfo, err := loadTexInfo(fs(h.Texinfo, data), mod.Textures)
		if err != nil {
			return nil, err
		}
		mod.TexInfos = texInfo
		sfaces, err := loadFacesV0(fs(h.Faces, data))
		if err != nil {
			return nil, err
		}
		msurfaces, err := buildSurfacesV0(sfaces, mod.Planes, mod.TexInfos)
		if err != nil {
			return nil, err
		}
		mod.Surfaces = msurfaces
		msurf, err := loadMarkSurfacesV0(fs(h.MarkSurfaces, data))
		if err != nil {
			return nil, err
		}
		mms, err := buildMarkSurfacesV0(msurf, mod.Surfaces)
		if err != nil {
			return nil, err
		}
		mod.MarkSurfaces = mms

		mod.VisData = fs(h.Visibility, data)
		leafs, err := loadLeafsV0(fs(h.Leafs, data))
		if err != nil {
			return nil, err
		}
		ml, err := buildLeafsV0(leafs, mod.MarkSurfaces, mod.VisData)
		if err != nil {
			return nil, err
		}
		mod.Leafs = ml
		nodes, err := loadNodesV0(fs(h.Nodes, data))
		if err != nil {
			return nil, err
		}
		mn, err := buildNodesV0(nodes, mod.Leafs, mod.Planes)
		if err != nil {
			return nil, err
		}
		mod.Nodes = mn

		scn, err := loadClipNodesV0(fs(h.ClipNodes, data))
		if err != nil {
			return nil, err
		}
		mcn, err := buildClipNodesV0(scn, mod.Planes)
		if err != nil {
			return nil, err
		}
		mod.ClipNodes = mcn

		mod.Entities = qm.ParseEntities(fs(h.Entities, data))

		submod, err := loadSubmodels(fs(h.Models, data))
		if err != nil {
			return nil, err
		}
		msm, err := buildSubmodels(submod)
		if err != nil {
			return nil, err
		}
		mod.Submodels = msm

		makeHulls(&mod.Hulls, mod.ClipNodes, mod.Planes, mod.Nodes)
		mod.FrameCount = 2

		mod.Node = mod.Nodes[0]

		// read 'submodels', submodel[0] is the 'map'
		// HeadNode [0] == first bsp node index
		// [1] == first clip node index
		// [2] == last clip node index
		// [3] usually 0
		for i, sub := range mod.Submodels {
			m := *mod
			if i > 0 {
				m.Name = fmt.Sprintf("*%d", i)
			}
			m.Hulls[0].FirstClipNode = sub.HeadNode[0]
			for j := 1; j < 4; j++ {
				m.Hulls[j].FirstClipNode = sub.HeadNode[j]
				m.Hulls[j].LastClipNode = len(mod.ClipNodes) - 1
			}
			// TODO
			// m.FirstModelSurface = sub.FirstFace
			// m.NumModelSurfaces = sub.FaceCount
			m.Mins = sub.Mins
			m.Maxs = sub.Maxs
			// TODO: calc rotate and yaw bounds
			// if i > 0 || mod.Name == SV_ModelName {
			// Why should this not be set for sv.worldmodel?
			m.ClipMins = sub.Mins
			m.ClipMaxs = sub.Maxs
			// }

			// VisLeafCount does not include the solid leaf 0, m.Leafs should still have it
			m.Leafs = m.Leafs[:sub.VisLeafCount+1]

			ret = append(ret, &m)
		}

	case bsp2Version2psb:
		log.Printf("Got V1 bsp: %v", h)
	case bsp2Versionbsp2:
		log.Printf("Got V2 bsp: %v", h)
	default:
		log.Printf("Version %v", h.Version)
	}

	return ret, nil
}

func buildPlanes(pls []*plane) []*qm.Plane {
	ret := make([]*qm.Plane, 0, len(pls))
	for _, pl := range pls {
		ret = append(ret, &qm.Plane{
			Normal: vec.Vec3{pl.Normal[0], pl.Normal[1], pl.Normal[2]},
			Dist:   pl.Distance,
			Type:   byte(pl.Type),
			SignBits: func() byte {
				r := 0
				for i := uint8(0); i < 3; i++ {
					if pl.Normal[i] < 0 {
						r |= 1 << i
					}
				}
				return byte(r)
			}(),
		})
	}
	return ret
}

func loadPlanes(data []byte) ([]*plane, error) {
	ret := []*plane{}
	buf := bytes.NewReader(data)
	for {
		val := &plane{}
		err := binary.Read(buf, binary.LittleEndian, val)
		switch err {
		default:
			return nil, err
		case io.EOF:
			return ret, nil
		case nil:
			ret = append(ret, val)
		}
	}
}

const (
	texSpecial = 1 << iota
	texMissing
)

func loadTexInfo(data []byte, textures []*qm.Texture) ([]*qm.TexInfo, error) {
	type texInfo struct {
		V      [2][4]float32
		MipTex uint32
		Flags  uint32
	}
	const texInfoSize = 40
	if len(data)%texInfoSize != 0 {
		return nil, fmt.Errorf("MOD_LoadBmodel: funny lump size")
	}
	buf := bytes.NewReader(data)
	count := len(data) / texInfoSize
	t := make([]*qm.TexInfo, count)

	missing := 0
	var ti texInfo
	for i := 0; i < count; i++ {
		err := binary.Read(buf, binary.LittleEndian, &ti)
		if err != nil {
			return nil, fmt.Errorf("loadTexInfo: %v", err)
		}
		qti := &qm.TexInfo{
			Vecs:  ti.V,
			Flags: ti.Flags,
		}
		// We added 2 textures in texture loading to handle missing ones here
		if int(ti.MipTex) < len(textures)-2 {
			qti.Texture = textures[ti.MipTex]
		} else {
			if ti.Flags&texSpecial != 0 {
				qti.Texture = textures[len(textures)-1]
			} else {
				qti.Texture = textures[len(textures)-2]
			}
			qti.Flags |= texMissing
			missing++
		}
		t[i] = qti
	}
	if missing > 0 {
		log.Printf("Mod_LoadTexinfo: %i texture(s) missing from BSP file", missing)
	}
	return t, nil
}

func buildMarkSurfacesV0(marks []int, surfaces []*qm.Surface) ([]*qm.Surface, error) {
	ret := make([]*qm.Surface, 0, len(marks))
	for _, m := range marks {
		if m >= len(surfaces) {
			return nil, fmt.Errorf("MarkSurfaces out of bounds")
		}
		ret = append(ret, surfaces[m])
	}
	return ret, nil
}

func loadFacesV0(data []byte) ([]*faceV0, error) {
	ret := []*faceV0{}
	buf := bytes.NewReader(data)
	for {
		val := &faceV0{}
		err := binary.Read(buf, binary.LittleEndian, val)
		switch err {
		default:
			return nil, err
		case io.EOF:
			return ret, nil
		case nil:
			ret = append(ret, val)
		}
	}
}

func buildSurfacesV0(f []*faceV0, plane []*qm.Plane, texinfo []*qm.TexInfo) ([]*qm.Surface, error) {
	// faceV0 {
	// PlaneID int16
	// Side int16
	// ListEdgeID int32
	// ListEdgeNumber int16
	// TexInfoID int16
	// LightStyle [4]uint8
	// LightMap int32
	// }
	ret := make([]*qm.Surface, 0, len(f))
	for range /*sf*/ f {
		nsf := &qm.Surface{
			// PlaneID int32
			// Side int32
			// ListEdgeID int32
			// ListEdgeNumber int32
			// TextInfoID int32
			// LightStyle [4]uint8
			// LightMap  int32
			//
			// TODO
			// firstedge
			// numedge
			// plane = plane[sf.planenum]
			// side
			// texinfo = textinfo[sf.textinfo]
			// styles
			// lightofs
			// flags = 0
			// if side != 0 {
			// flags |= SURF_PLANEBACK
		}
		// calcsurfaceExtends
		// caldSurfaceBounds
		ret = append(ret, nsf)
	}
	return ret, nil
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
	ret := make([]*qm.MLeaf, 0, len(ls))
	for _, l := range ls {
		nv := func() []byte {
			if l.VisOfs == -1 {
				return nil
			}
			return vd[l.VisOfs:]
		}()
		nl := &qm.MLeaf{
			NodeBase: qm.NewNodeBase(int(l.Type), 0, [6]float32{
				float32(l.Box[0]), float32(l.Box[1]), float32(l.Box[2]),
				float32(l.Box[3]), float32(l.Box[4]), float32(l.Box[5])}),
			CompressedVis:     nv,
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
	ret := make([]*qm.MNode, 0, len(nd))
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
	getChild := func(id uint16) qm.Node {
		if int(id) < len(ret) {
			return ret[id]
		}
		p := 65535 - int(id) // this is intentionally, -1 is leaf 0
		if p < len(leafs) {
			return leafs[p]
		}
		log.Printf("No Child. Got child id %d of %d, p %d of %d. ",
			id, len(ret), p, len(leafs))
		return nil
	}
	for i, n := range nd {
		ret[i].Children[0] = getChild(n.Children[0])
		ret[i].Children[1] = getChild(n.Children[1])
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

func buildClipNodesV0(scns []*clipNodeV0, pls []*qm.Plane) ([]*qm.ClipNode, error) {
	ret := make([]*qm.ClipNode, 0, len(scns))
	for _, scn := range scns {
		if scn.PlaneNumber < 0 || int(scn.PlaneNumber) >= len(pls) {
			return nil, fmt.Errorf("buildClipNodesV0, planenum out of bounds")
		}
		cn := &qm.ClipNode{
			Plane:    pls[int(scn.PlaneNumber)],
			Children: [2]int{int(scn.Children[0]), int(scn.Children[1])},
		}
		if cn.Children[0] >= len(scns) {
			cn.Children[0] -= 65536
		}
		if cn.Children[1] >= len(scns) {
			cn.Children[1] -= 65536
		}
		ret = append(ret, cn)
	}
	return ret, nil
}

func loadClipNodesV0(data []byte) ([]*clipNodeV0, error) {
	ret := []*clipNodeV0{}
	buf := bytes.NewReader(data)
	for {
		n := &clipNodeV0{}
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

// func loadEntities(fs(h.Entities , data),ret)
// func loadModes(fs(h.Models , data),ret)
// func makeHull0(ret) {
func makeHulls(hs *[4]qm.Hull, cns []*qm.ClipNode, pns []*qm.Plane, ns []*qm.MNode) {
	hs[0].ClipNodes = make([]*qm.ClipNode, 0, len(ns))

	getNodeNum := func(qn qm.Node) int {
		node, ok := qn.(*qm.MNode)
		if !ok {
			return qn.Contents()
		}
		for i, n := range ns {
			if n == node {
				return i
			}
		}
		log.Printf("Could not find node number")
		return -1
	}

	for _, cn := range ns {
		hs[0].ClipNodes = append(hs[0].ClipNodes, &qm.ClipNode{
			Plane:    cn.Plane,
			Children: [2]int{getNodeNum(cn.Children[0]), getNodeNum(cn.Children[1])},
		})
	}
	hs[0].FirstClipNode = 0
	hs[0].LastClipNode = len(ns) - 1
	hs[0].Planes = pns
	// hs[0].ClipMins?
	// hs[0].ClipMaxs?

	hs[1].ClipMins = vec.Vec3{-16, -16, -24}
	hs[1].ClipMaxs = vec.Vec3{16, 16, 32}
	hs[1].ClipNodes = cns
	hs[1].FirstClipNode = 0
	hs[1].LastClipNode = len(cns) - 1
	hs[1].Planes = pns

	hs[2].ClipMins = vec.Vec3{-32, -32, -24}
	hs[2].ClipMaxs = vec.Vec3{32, 32, 64}
	hs[2].ClipNodes = cns
	hs[2].FirstClipNode = 0
	hs[2].LastClipNode = len(cns) - 1
	hs[2].Planes = pns
}

func loadSubmodels(data []byte) ([]*model, error) {
	ret := []*model{}
	buf := bytes.NewReader(data)
	for {
		m := &model{}
		err := binary.Read(buf, binary.LittleEndian, m)
		switch err {
		default:
			return nil, err
		case io.EOF:
			return ret, nil
		case nil:
			ret = append(ret, m)
		}
	}
}

func buildSubmodels(mod []*model) ([]*qm.Submodel, error) {
	if len(mod) == 0 {
		return nil, fmt.Errorf("No model found")
	}
	if mod[0].VisLeafCount > 70000 {
		return nil, fmt.Errorf(
			"LoadSubModels: too many visleafs (%d, max = %d)",
			mod[0].VisLeafCount, 70000)
	}
	if mod[0].VisLeafCount > 8192 {
		log.Printf("%d visleafs exceeds standard limit of 8192", mod[0].VisLeafCount)
	}
	ret := make([]*qm.Submodel, 0, len(mod))
	for _, m := range mod {
		ret = append(ret, &qm.Submodel{
			// Therjak: orig reduces mins and extends max by 1, here it breaks stuff. Why?
			Mins:   vec.Vec3{m.BoundingBox[0] - 1, m.BoundingBox[1] - 1, m.BoundingBox[2] - 1},
			Maxs:   vec.Vec3{m.BoundingBox[3] + 1, m.BoundingBox[4] + 1, m.BoundingBox[5] + 1},
			Origin: vec.Vec3{m.Origin[0], m.Origin[1], m.Origin[2]},
			HeadNode: [4]int{
				int(m.HeadNode[0]), int(m.HeadNode[1]), int(m.HeadNode[2]), int(m.HeadNode[3]),
			},
			VisLeafCount: int(m.VisLeafCount),
			FirstFace:    int(m.FirstFace),
			FaceCount:    int(m.FaceCount),
		})
	}
	return ret, nil
}

var (
	noTextureMip = &qm.Texture{
		Name:   "notexture",
		Width:  32,
		Height: 32,
	}
	noTextureMip2 = &qm.Texture{
		Name:   "notexture2",
		Width:  32,
		Height: 32,
	}
)

func loadTextures(data []byte) ([]*qm.Texture, error) {
	numTex := int32(0)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &numTex)
	if err != nil || numTex == 0 {
		return nil, nil
	}
	// Need 2 dummy textures to handle missing ones
	t := make([]*qm.Texture, numTex+2)
	for i := int32(0); i < numTex; i++ {
		t[i] = &qm.Texture{}
	}
	t[len(t)-1] = noTextureMip  // lightmapped surfs
	t[len(t)-2] = noTextureMip2 // SURF_DRAWTILED surfs

	// Texture {
	// Width int
	// Height int
	// Name string
	// TextureChains [2]*Surface
	// Texture
	// Fullbright
	// Warp }
	return t, nil
}

func loadEdgesV0(data []byte) ([]*qm.MEdge, error) {
	type dsedge struct {
		V [2]uint16
	}
	const dsedgeSize = 4
	if len(data)%dsedgeSize != 0 {
		return nil, fmt.Errorf("MOD_LoadBmodel: funny lump size")
	}
	buf := bytes.NewReader(data)
	count := len(data) / dsedgeSize
	t := make([]*qm.MEdge, count)
	var dedge dsedge
	for i := 0; i < count; i++ {
		err := binary.Read(buf, binary.LittleEndian, &dedge)
		if err != nil {
			return nil, fmt.Errorf("loadEdgesV0: %v", err)
		}
		edge := &qm.MEdge{}
		edge.V[0] = int(dedge.V[0])
		edge.V[1] = int(dedge.V[1])
		t[i] = edge
	}
	return t, nil
}

func loadEdgesV2(data []byte) ([]*qm.MEdge, error) {
	type dledge struct {
		V [2]uint32
	}
	const dledgeSize = 8
	if len(data)%dledgeSize != 0 {
		return nil, fmt.Errorf("MOD_LoadBmodel: funny lump size")
	}
	buf := bytes.NewReader(data)
	count := len(data) / dledgeSize
	t := make([]*qm.MEdge, count)
	var dedge dledge
	for i := 0; i < count; i++ {
		err := binary.Read(buf, binary.LittleEndian, &dedge)
		if err != nil {
			return nil, fmt.Errorf("loadEdgesV2: %v", err)
		}
		edge := &qm.MEdge{}
		edge.V[0] = int(dedge.V[0])
		edge.V[1] = int(dedge.V[1])
		t[i] = edge
	}
	return t, nil
}

func loadVertexes(data []byte) ([]*qm.MVertex, error) {
	type dvertex struct {
		Point [3]float32
	}
	const dvertexSize = 12
	if len(data)%dvertexSize != 0 {
		return nil, fmt.Errorf("MOD_LoadBmodel: funny lump size")
	}
	buf := bytes.NewReader(data)
	count := len(data) / dvertexSize
	t := make([]*qm.MVertex, count)
	var dv dvertex
	for i := 0; i < count; i++ {
		err := binary.Read(buf, binary.LittleEndian, &dv)
		if err != nil {
			return nil, fmt.Errorf("loadVertexes: %v", err)
		}
		v := &qm.MVertex{}
		v.Position[0] = dv.Point[0]
		v.Position[1] = dv.Point[1]
		v.Position[2] = dv.Point[2]
		t[i] = v
	}
	return t, nil

}
