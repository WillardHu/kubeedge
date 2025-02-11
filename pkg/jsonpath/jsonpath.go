/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonpath

import (
	"encoding/json"
)

const (
	OpAdd     = "add"
	OpRemove  = "remove"
	OpReplace = "replace"
)

type Operation string

type Object struct {
	Op    Operation `json:"op"`
	Path  string    `json:"path"`
	Value string    `json:"value,omitempty"`
}

func New(op Operation, path string) Object {
	return Object{
		Op:   op,
		Path: path,
	}
}

func (o *Object) SetValue(value any) error {
	if value == nil {
		return nil
	}
	switch value := value.(type) {
	case string:
		o.Value = value
	default:
		bff, err := json.Marshal(value)
		if err != nil {
			return err
		}
		o.Value = string(bff)
	}
	return nil
}

func (o *Object) ToJSON() ([]byte, error) {
	return json.Marshal(o)
}
