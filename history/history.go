// SPDX-License-Identifier: GPL-2.0-or-later

package history

import (
	"fmt"
	"os"
	"path/filepath"

	"goquake/filesystem"
	"goquake/protos"

	"google.golang.org/protobuf/proto"
)

const (
	// add a max size to prevent the file from growing indefinitely
	maxHistory = 32
)

type History struct {
	txt []string
	idx int
}

func (h *History) String() string {
	if len(h.txt) == h.idx {
		return ""
	}
	return h.txt[h.idx]
}

func (h *History) Up() {
	if h.idx > 0 {
		h.idx--
	}
}

func (h *History) Down() {
	if h.idx < len(h.txt) {
		h.idx++
	}
}

func (h *History) Add(s string) {
	h.txt = append(h.txt, s)
	h.idx = len(h.txt)
}

const (
	historyFilename = "history.txt"
)

func (h *History) Load() error {
	fullname := filepath.Join(filesystem.BaseDir(), historyFilename)
	in, err := os.ReadFile(fullname)
	if err != nil {
		// assume no history file
		return nil
	}
	data := &protos.History{}
	if err := proto.Unmarshal(in, data); err != nil {
		return fmt.Errorf("failed to decode history")
	}
	h.txt = data.GetEntries()
	h.idx = len(h.txt)
	return nil
}

func (h *History) Save() error {
	fullname := filepath.Join(filesystem.BaseDir(), historyFilename)
	l := min(len(h.txt), maxHistory)
	data := protos.History_builder{
		Entries: h.txt[:l],
	}.Build()
	out, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to encode history")
	}
	if err := os.WriteFile(fullname, out, 0660); err != nil {
		return fmt.Errorf("failed to write history file")
	}
	return nil
}
