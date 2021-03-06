// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package merge

import (
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/suite"
)

func TestThreeWayMapMerge(t *testing.T) {
	suite.Run(t, &ThreeWayMapMergeSuite{})
}

func TestThreeWayStructMerge(t *testing.T) {
	suite.Run(t, &ThreeWayStructMergeSuite{})
}

type kvs []interface{}

func (kv kvs) items() []interface{} {
	return kv
}

func (kv kvs) remove(k interface{}) kvs {
	out := make(kvs, 0, len(kv))
	for i := 0; i < len(kv); i += 2 {
		if kv[i] != k {
			out = append(out, kv[i], kv[i+1])
		}
	}
	return out
}

func (kv kvs) set(k, v interface{}) kvs {
	out := make(kvs, len(kv))
	for i := 0; i < len(kv); i += 2 {
		out[i], out[i+1] = kv[i], kv[i+1]
		if kv[i] == k {
			out[i+1] = v
		}
	}
	return out
}

var (
	aa1      = kvs{"a1", "a-one", "a2", "a-two", "a3", "a-three", "a4", "a-four"}
	aa1a     = kvs{"a1", "a-one", "a2", "a-two", "a3", "a-three-diff", "a4", "a-four", "a6", "a-six"}
	aa1b     = kvs{"a1", "a-one", "a3", "a-three-diff", "a4", "a-four", "a5", "a-five"}
	aaMerged = kvs{"a1", "a-one", "a3", "a-three-diff", "a4", "a-four", "a5", "a-five", "a6", "a-six"}

	mm1       = kvs{}
	mm1a      = kvs{"k1", kvs{"a", 0}}
	mm1b      = kvs{"k1", kvs{"b", 1}}
	mm1Merged = kvs{"k1", kvs{"a", 0, "b", 1}}

	mm2       = kvs{"k2", aa1, "k3", "k-three"}
	mm2a      = kvs{"k1", kvs{"a", 0}, "k2", aa1a, "k3", "k-three", "k4", "k-four"}
	mm2b      = kvs{"k1", kvs{"b", 1}, "k2", aa1b}
	mm2Merged = kvs{"k1", kvs{"a", 0, "b", 1}, "k2", aaMerged, "k4", "k-four"}
)

type ThreeWayKeyValMergeSuite struct {
	ThreeWayMergeSuite
}

type ThreeWayMapMergeSuite struct {
	ThreeWayKeyValMergeSuite
}

func (s *ThreeWayMapMergeSuite) SetupSuite() {
	s.create = func(seq seq) (val types.Value) {
		if seq != nil {
			keyValues := valsToTypesValues(s.create, seq.items()...)
			val = types.NewMap(keyValues...)
		}
		return
	}
	s.typeStr = "Map"
}

type ThreeWayStructMergeSuite struct {
	ThreeWayKeyValMergeSuite
}

func (s *ThreeWayStructMergeSuite) SetupSuite() {
	s.create = func(seq seq) (val types.Value) {
		if seq != nil {
			kv := seq.items()
			fields := types.StructData{}
			for i := 0; i < len(kv); i += 2 {
				fields[kv[i].(string)] = valToTypesValue(s.create, kv[i+1])
			}
			val = types.NewStruct("TestStruct", fields)
		}
		return
	}
	s.typeStr = "struct"
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_DoNothing() {
	s.tryThreeWayMerge(nil, nil, aa1, aa1, nil)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NoRecursion() {
	s.tryThreeWayMerge(aa1a, aa1b, aa1, aaMerged, nil)
	s.tryThreeWayMerge(aa1b, aa1a, aa1, aaMerged, nil)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveCreate() {
	s.tryThreeWayMerge(mm1a, mm1b, mm1, mm1Merged, nil)
	s.tryThreeWayMerge(mm1b, mm1a, mm1, mm1Merged, nil)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveCreateNil() {
	s.tryThreeWayMerge(mm1a, mm1b, nil, mm1Merged, nil)
	s.tryThreeWayMerge(mm1b, mm1a, nil, mm1Merged, nil)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveMerge() {
	s.tryThreeWayMerge(mm2a, mm2b, mm2, mm2Merged, nil)
	s.tryThreeWayMerge(mm2b, mm2a, mm2, mm2Merged, nil)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RefMerge() {
	vs := types.NewTestValueStore()

	strRef := vs.WriteValue(types.NewStruct("Foo", types.StructData{"life": types.Number(42)}))

	m := kvs{"r2", vs.WriteValue(s.create(aa1))}
	ma := kvs{"r1", strRef, "r2", vs.WriteValue(s.create(aa1a))}
	mb := kvs{"r1", strRef, "r2", vs.WriteValue(s.create(aa1b))}
	mMerged := kvs{"r1", strRef, "r2", vs.WriteValue(s.create(aaMerged))}
	vs.Flush()

	s.tryThreeWayMerge(ma, mb, m, mMerged, vs)
	s.tryThreeWayMerge(mb, ma, m, mMerged, vs)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_RecursiveMultiLevelMerge() {
	vs := types.NewTestValueStore()

	m := kvs{"mm1", mm1, "mm2", vs.WriteValue(s.create(mm2))}
	ma := kvs{"mm1", mm1a, "mm2", vs.WriteValue(s.create(mm2a))}
	mb := kvs{"mm1", mm1b, "mm2", vs.WriteValue(s.create(mm2b))}
	mMerged := kvs{"mm1", mm1Merged, "mm2", vs.WriteValue(s.create(mm2Merged))}
	vs.Flush()

	s.tryThreeWayMerge(ma, mb, m, mMerged, vs)
	s.tryThreeWayMerge(mb, ma, m, mMerged, vs)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NilConflict() {
	s.tryThreeWayConflict(nil, s.create(mm2b), s.create(mm2), "Cannot merge nil Value with")
	s.tryThreeWayConflict(s.create(mm2a), nil, s.create(mm2), "with nil value.")
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_ImmediateConflict() {
	s.tryThreeWayConflict(types.NewSet(), s.create(mm2b), s.create(mm2), "Cannot merge Set<> with "+s.typeStr)
	s.tryThreeWayConflict(s.create(mm2b), types.NewSet(), s.create(mm2), "Cannot merge "+s.typeStr)
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NestedConflict() {
	a := mm2a.set("k2", types.NewSet())
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), types.EncodedValue(types.NewSet()))
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), types.EncodedValue(s.create(aa1b)))
}

func (s *ThreeWayKeyValMergeSuite) TestThreeWayMerge_NestedConflictingOperation() {
	a := mm2a.remove("k2")
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), `removed "k2"`)
	s.tryThreeWayConflict(s.create(a), s.create(mm2b), s.create(mm2), `modded "k2"`)
}
