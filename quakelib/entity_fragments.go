package quakelib

import (
	"log"

	"github.com/therjak/goquake/bsp"
	"github.com/therjak/goquake/math/vec"
)

type entityFragment struct {
	// leaf is used to remove the efrag from the leaf
	leaf *bsp.MLeaf
	// the linked list for efrags belonging to the same leaf
	leafNext *entityFragment
	entity   *Entity
	//
	entNext *entityFragment
}

var (
	// assumptions for efrags are:
	// each leaf only contains a small number of efrags
	// slice of pointers is equally random access to linked list || this is probably false
	// adding and removing efrags happen often
	efrags     [4096]entityFragment
	freeEfrags *entityFragment
)

func clearEntityFragments() {
	freeEfrags = &efrags[0]
	for i := 0; i < len(efrags)-2; i++ {
		efrags[i].entNext = &efrags[i+1]
	}
	efrags[len(efrags)-1].entNext = nil
}

func RemoveEntityFragments(e *Entity) {
	ef := e.Fragment
	for ef != nil { // run though the entityFragments on the Entity
		head := ef.leaf.Temporary.(*entityFragment)
		if head == ef {
			ef.leaf.Temporary = ef.leafNext
		} else {
			for { // run through the leafs of the entityFragment
				next := head.leafNext
				if next == nil {
					log.Printf("RemoveEntityFragments: fragment not found")
					break
				}
				if next == ef {
					head.leafNext = next.leafNext
					break
				}
				head = next
			}
		}

		old := ef
		ef = ef.entNext

		old.entNext = freeEfrags
		freeEfrags = old
	}

	e.Fragment = nil
}

type EntityFragmentAdder struct {
	entity   *Entity
	world    *bsp.Model
	mins     vec.Vec3
	maxs     vec.Vec3
	lastLink **entityFragment
}

func (e *EntityFragmentAdder) Do() {
	m := e.entity.Model
	if m == nil {
		// noting to show so do not bother
		return
	}
	e.lastLink = &e.entity.Fragment
	e.mins = vec.Add(e.entity.Origin, m.Mins())
	e.maxs = vec.Add(e.entity.Origin, m.Maxs())
	e.splitOnNode(e.world.Node)
}

func (e *EntityFragmentAdder) splitOnNode(node bsp.Node) {
	switch n := node.(type) {
	case *bsp.MLeaf:
		ef := freeEfrags
		if ef == nil {
			// no free fragments
			return
		}
		freeEfrags = ef.entNext
		ef.entity = e.entity

		// add entityFragment to entity
		*e.lastLink = ef
		e.lastLink = &ef.entNext
		ef.entNext = nil

		// add entityFragment to leaf
		ef.leaf = n
		// *entityFragment and nil are both ok
		ef.leafNext, _ = n.Temporary.(*entityFragment)
		n.Temporary = ef

	case *bsp.MNode:
		sides := boxOnPlaneSide(e.mins, e.maxs, n.Plane)
		if sides&1 != 0 {
			e.splitOnNode(n.Children[0])
		}
		if sides&2 != 0 {
			e.splitOnNode(n.Children[1])
		}
	}
}

func MakeEntitiesVisible(leaf *bsp.MLeaf) {
	ef := leaf.Temporary.(*entityFragment)
	for ef != nil {
		ent := ef.entity
		if ent.VisFrame != R_framecount() && VisibleEntitiesNum() < 4096 {
			ent.VisFrame = R_framecount()
			cl.AddVisibleEntity(ent)
		}
		ef = ef.leafNext
	}
}
