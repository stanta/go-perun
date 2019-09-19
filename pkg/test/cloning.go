// Copyright (c) 2019 The Perun Authors. All rights reserved.
// This file is part of go-perun. Use of this source code is governed by a
// MIT-style license that can be found in the LICENSE file.

package test // import "perun.network/go-perun/pkg/test"

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"testing"
)


// For the given type, this function checks if it possesses a method `Clone`.
// Receiver and return value can be values or references, e.g., with a method
// `func (*T) Clone() T`, the type `T` is considered cloneable.
func isCloneable(t reflect.Type) bool {
	kind := t.Kind()

	if kind != reflect.Struct && kind != reflect.Ptr {
		return false
	}

	// t may be a **struct
	baseType := t
	ptrType := reflect.PtrTo(t)

	for baseType.Kind() == reflect.Ptr {
		ptrType = baseType
		baseType = ptrType.Elem()
	}


	// check for clone method
	method, ok := ptrType.MethodByName("Clone")

	if !ok {
		return ok
	}

	methodType := method.Type
	numIn := methodType.NumIn()

	if numIn != 1 {
		return false
	}

	inputType := methodType.In(0)

	if inputType != ptrType {
		return false
	}

	numOut := methodType.NumOut()

	if numOut != 1 {
		return false
	}

	outputType := methodType.Out(0)

	if outputType != baseType && outputType != ptrType {
		return false
	}

	return true
}



func checkCloneImpl(v, w reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		if v.Pointer() == 0 && w.Pointer() == 0 {
			return nil
		}

		if v.Pointer() == 0 && w.Pointer() != 0 {
			return errors.New("First pointer is nil, second pointer is non-nil")
		}

		if v.Pointer() != 0 && w.Pointer() == 0 {
			return errors.New("First pointer is non-nil, second pointer is nil")
		}

		if v.Pointer() == w.Pointer() {
			return errors.New("Both arguments reference the same structure")
		}
	}

	if v.Kind() == reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		log.Fatalf("BUG: expected reference to struct, got reference to reference")
	}


	// get struct type
	baseType := v.Type()
	ptrType := reflect.PtrTo(baseType)

	if baseType.Kind() == reflect.Ptr {
		v = v.Elem()
		w = w.Elem()
		ptrType = baseType
		baseType = ptrType.Elem()
	}

	t := baseType


	// check for field tags
	for i := 0; i < baseType.NumField(); i++ {
		f := baseType.Field(i)

		kind := f.Type.Kind()
		left := v.Field(i)
		right := w.Field(i)

		// disallow some untested kinds
		if kind == reflect.Chan ||
			kind == reflect.Func || // disallow because of caputered references
			kind == reflect.Map ||
			kind == reflect.String ||
			kind == reflect.UnsafePointer {
			log.Fatalf("Implementation not tested with %v", kind)
		}

		tag, hasTag := f.Tag.Lookup("cloneable")

		// find unknown and misplaced tags
		if hasTag {
			if tag == "shallow" {
				if kind != reflect.Ptr && kind != reflect.Slice {
					format :=
						"Expected field %v.%s with tag '%s' to be a " +
						"pointer or a slice, got kind %v"
					return fmt.Errorf(format, t, f.Name, tag, kind)
				}
			} else if tag == "shallowElements" {
				if kind != reflect.Array && kind != reflect.Slice {
					format :=
						"Expected field %v.%s with tag '%s' to be an array or "+
						"a slice, got kind %v"
					return fmt.Errorf(format, t, f.Name, tag, kind)
				}
			} else {
				format := `Unknown tag 'cloneable:"%s"' on field %v.%s`
				return fmt.Errorf(format, tag, t, f.Name)
			}
		}

		// check actual field contents
		if kind == reflect.Ptr || kind == reflect.Slice {
			p := left.Pointer()
			q := right.Pointer()

			if p != q && hasTag && tag == "shallow" {

				format :=
					"Expected fields %v.%s with tag '%s' to have same pointees"
				return fmt.Errorf(format, t, f.Name, tag)
			}

			// the length check below is necessary because all slices created
			// empty seem to reference the same address in memory
			if p == q && p != 0 &&
				(!hasTag || tag != "shallow") &&
				(kind == reflect.Ptr || left.Len() > 0) {
				format := "Expected fields %v.%s to have different pointees"
				return fmt.Errorf(format, t, f.Name)
			}
		}

		if kind == reflect.Array || kind == reflect.Slice {
			n := left.Len()

			for j := 0; j < n; j++ {
				kind_j := left.Index(j).Kind()

				if kind_j == reflect.Ptr || kind_j == reflect.Slice {
					p := left.Index(j).Pointer()
					q := right.Index(j).Pointer()

					if p != q && hasTag && tag == "shallowElements" {
						format :=
							"Expected elements %v.%s[%d] in slices with tag " +
							"'%s' to have same pointees"
						return fmt.Errorf(format, t, f.Name, j, tag)
					}

					if p == q && p != 0 && (!hasTag || tag != "shallowElements") {
						format :=
							"Expected elements %v.%s[%d] to have different pointees"
						return fmt.Errorf(format, t, f.Name, j)
					}
				} else if kind_j == reflect.Struct && isCloneable(f.Type.Elem()) {
					err := checkCloneImpl(left.Index(j), right.Index(j))

					if err != nil {
						format := "Error in cloneable element %v.%s[%d]: %v"
						return fmt.Errorf(format, t, f.Name, j, err)
					}
				}
			}
		} else if kind == reflect.Struct && isCloneable(f.Type) {
			err := checkCloneImpl(left, right)

			if err != nil {
				return err
			}
		}
	}

	return nil
}




// Given two values, this function checks if they could be clones.
// This implementation is incomplete:
// * Slices are not checked.
// * Not exported fields are ignored.
// * Pointer equality is not checked meaning with `checkClone(&x,&x)` will not
//   return an error when it should.
func checkClone(p, q interface{}) error {
	if !reflect.DeepEqual(p, q) {
		return errors.New("Proper clones must be deeply equal")
	}

	if !isCloneable(reflect.TypeOf(p)) {
		return errors.New("First argument must be cloneable")
	}

	if !isCloneable(reflect.TypeOf(q)) {
		return errors.New("Second argument must be cloneable")
	}

	v := reflect.ValueOf(p)
	w := reflect.ValueOf(q)

	return checkCloneImpl(v, w)
}



// Given `x`, call `x.Clone()` if possible, return an error otherwise.
func clone(x interface{}) (interface{}, error) {
	if x == nil {
		return nil, fmt.Errorf("Cannot clone nil reference")
	}
	if !isCloneable(reflect.TypeOf(x)) {
		return nil, fmt.Errorf("Input of type %T is not cloneable", x)
	}

	v := reflect.ValueOf(x)

	if v.Kind() != reflect.Ptr && v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Expected pointer or struct, got %v", v.Kind())
	}

	if clone := v.MethodByName("Clone"); clone.IsValid() {
		// num return values is checked by `isCloneable`
		return clone.Call([]reflect.Value{})[0].Interface(), nil
	}

	return nil, fmt.Errorf("Type %T does not possess a Clone() method", x)
}



// This function attemps to recognize improper cloning.
// Initially, this function will clone its input `x` by calling `x.Clone()`,
// where `x` is an instance of a struct (or a reference). Then it attemps to
// detect improper clones by taking the following steps for every exported
// field of `x`:
// * If the field of type `T` is itself is a cloneable, then this function is
//   called on the field value.
// * If the field has a `clonable:"shallow"` tag, it is checked that the
//   pointer or slice value are the same. If the type of the field is a value
//   type, the test fails immediately with the error that a value field cannot
//   have this tag.
// * If the field has a `clonable:"shallowElements"` tag, it is checked that the
//   slice itself is different but that the slice values are the same. If the
//   field is not a slice value, the test fails immediately.
// * Otherwise, the field is tested with `reflect.DeepEqual`. Missing fields
// cause an error.
func VerifyClone(t* testing.T, x interface{}) {
	if !isCloneable(reflect.TypeOf(x)) {
		t.Errorf("Expected cloneable input, got %v (type %T)", x, x)
	}

	c, err := clone(x)

	if err != nil {
		t.Errorf("Cloning failure: %v", err)
	}

	if err = checkClone(x, c); err != nil {
		t.Error(err)
	}
}
