package mdl

import (
	qm "quake/model"
)

func Load(name string, data []byte) ([]*qm.QModel, error) {
	var ret []*qm.QModel
	mod := &qm.QModel{
		Name: name,
		Type: qm.ModAlias,
	}

	// TODO: load the actual model

	ret = append(ret, mod)
	return ret, nil
}
