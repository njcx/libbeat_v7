// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package winevent

import (
	"fmt"
	"reflect"

	"github.com/njcx/libbeat_v7/common"
	"github.com/njcx/libbeat_v7/processors/decode_xml_wineventlog/sys"
)

// AddOptional adds a key and value to the given MapStr if the value is not the
// zero value for the type of v. It is safe to call the function with a nil
// MapStr.
func AddOptional(m common.MapStr, key string, v interface{}) {
	if m != nil && !isZero(v) {
		_, _ = m.Put(key, v)
	}
}

// AddPairs adds a new dictionary to the given MapStr. The key/value pairs are
// added to the new dictionary. If any keys are duplicates, the first key/value
// pair is added and the remaining duplicates are dropped. Pair keys are not
// expanded into dotted paths.
//
// The new dictionary is added to the given MapStr and it is also returned for
// convenience purposes.
func AddPairs(m common.MapStr, key string, pairs []KeyValue) common.MapStr {
	if len(pairs) == 0 {
		return nil
	}

	// Explicitly use the unnamed type to prevent accidental use
	// of common.MapStr path look-up methods.
	h := make(map[string]interface{}, len(pairs))

	for i, kv := range pairs {
		// Ignore empty values.
		if kv.Value == "" {
			continue
		}

		// If the key name is empty or if it the default of "Data" then
		// assign a generic name of paramN.
		k := kv.Key
		if k == "" || k == "Data" {
			k = fmt.Sprintf("param%d", i+1)
		}

		// Do not overwrite.
		_, exists := h[k]
		if exists {
			debugf("Dropping key/value (k=%s, v=%s) pair because key already "+
				"exists. event=%+v", k, kv.Value, m)
		} else {
			h[k] = sys.RemoveWindowsLineEndings(kv.Value)
		}
	}

	if len(h) == 0 {
		return nil
	}

	_, _ = m.Put(key, common.MapStr(h))

	return h
}

// isZero return true if the given value is the zero value for its type.
func isZero(i interface{}) bool {
	if i == nil {
		return true
	}

	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}

	return false
}
