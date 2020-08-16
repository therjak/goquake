package model

import (
	"bytes"
	"encoding/json"
	"log"
)

type Entity struct {
	properties map[string]string
}

func NewEntity(p map[string]string) *Entity {
	return &Entity{p}
}

func (e *Entity) Property(name string) (string, bool) {
	v, ok := e.properties[name]
	return v, ok
}

func (e *Entity) Name() (string, bool) {
	v, ok := e.properties["classname"]
	return v, ok
}

func (e *Entity) PropertyNames() []string {
	n := []string{}
	for k, _ := range e.properties {
		n = append(n, k)
	}
	return n
}

func ParseEntities(data []byte) []*Entity {
	/*
		The data looks like:
			{
				"name" "value"
				"name2" "value2"
			}
			{
				"name3" "value"
				{
					()()()...
				}
			}
	*/
	data = bytes.ReplaceAll(data, []byte("\" \""), []byte("\": \""))
	data = bytes.ReplaceAll(data, []byte("\n"), []byte{})
	data = bytes.ReplaceAll(data, []byte("\"\""), []byte("\",\""))
	data = bytes.ReplaceAll(data, []byte("}{"), []byte("},{"))
	// Stupid workaround for escaping. Can happen in windows paths. Does probably not catch 100%
	// Would be preferred to skip escaping in the JSON decoder.
	// For now just 'fix' the 5 cases known inside the decoder.
	data = bytes.ReplaceAll(data, []byte("\\b"), []byte("\b"))
	data = bytes.ReplaceAll(data, []byte("\\f"), []byte("\f"))
	data = bytes.ReplaceAll(data, []byte("\\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\\r"), []byte("\r"))
	data = bytes.ReplaceAll(data, []byte("\\t"), []byte("\t"))
	data = bytes.ReplaceAll(data, []byte("\\"), []byte("/"))
	data = bytes.ReplaceAll(data, []byte("\b"), []byte("\\b"))
	data = bytes.ReplaceAll(data, []byte("\f"), []byte("\\f"))
	data = bytes.ReplaceAll(data, []byte("\n"), []byte("\\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\\r"))
	data = bytes.ReplaceAll(data, []byte("\t"), []byte("\\t"))
	j := make([]byte, len(data)+1)
	copy(j[1:], data)
	j[0] = '['
	j[len(j)-1] = ']'
	var result []map[string]string
	err := json.Unmarshal(j, &result)
	if err != nil {
		log.Printf("unmarshal err: %v", err)
	}
	es := []*Entity{}
	for _, m := range result {
		es = append(es, NewEntity(m))
	}
	return es
}
