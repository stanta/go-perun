// Copyright (c) 2019 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of go-perun. Use
// of this source code is governed by a MIT-style license that can be found in
// the LICENSE file.

package test // import "perun.network/go-perun/pkg/test"

import (
	"math/rand"
	"testing"

	"perun.network/go-perun/pkg/io"
	"perun.network/go-perun/wire"

	"github.com/stretchr/testify/assert"
)

// TestEqualEncoding tests EqualEncoding
func TestEqualEncoding(t *testing.T) {
	rng := rand.New(rand.NewSource(1337))
	a := make(wire.ByteSlice, 10)
	b := make(wire.ByteSlice, 10)
	c := make(wire.ByteSlice, 12)

	rng.Read(a)
	rng.Read(b)
	rng.Read(c)
	c2 := wire.ByteSlice(c)

	tests := []struct {
		a         io.Encoder
		b         io.Encoder
		shouldOk  bool
		shouldErr bool
		name      string
	}{
		{a, nil, false, true, "one Encoder set to nil"},
		{nil, a, false, true, "one Encoder set to nil"},
		{io.Encoder(nil), b, false, true, "one Encoder set to nil"},
		{b, io.Encoder(nil), false, true, "one Encoder set to nil"},

		{nil, nil, true, false, "both Encoders set to nil"},
		{io.Encoder(nil), io.Encoder(nil), true, false, "both Encoders set to nil"},

		{a, a, true, false, "same Encoders"},
		{a, &a, true, false, "same Encoders"},
		{&a, a, true, false, "same Encoders"},
		{&a, &a, true, false, "same Encoders"},

		{c, c2, true, false, "different Encoders and same content"},

		{a, b, false, false, "different Encoders and different content"},
		{a, c, false, false, "different Encoders and different content"},
	}

	for _, tt := range tests {
		ok, err := EqualEncoding(tt.a, tt.b)

		assert.Equalf(t, ok, tt.shouldOk, "EqualEncoding with %s should return %t as bool but got: %t", tt.name, tt.shouldOk, ok)
		assert.Falsef(t, (err == nil) && tt.shouldErr, "EqualEncoding with %s should return an error but got nil", tt.name)
		assert.Falsef(t, (err != nil) && !tt.shouldErr, "EqualEncoding with %s should return nil as error but got: %s", tt.name, err)
	}
}
