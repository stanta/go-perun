// Copyright (c) 2019 Chair of Applied Cryptography, Technische Universität
// Darmstadt, Germany. All rights reserved. This file is part of go-perun. Use
// of this source code is governed by a MIT-style license that can be found in
// the LICENSE file.

package channel

import (
	"context"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/channel"
	wallettest "perun.network/go-perun/wallet/test"

	"perun.network/go-perun/backend/ethereum/channel/test"
	ethwallettest "perun.network/go-perun/backend/ethereum/wallet/test"
	channeltest "perun.network/go-perun/channel/test"
	perunwallet "perun.network/go-perun/wallet"
)

const (
	keyDir   = "../wallet/testdata"
	password = "secret"
)

const timeout = 20 * time.Second

func TestFunder_Fund(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	f := newSimulatedFunder(t)
	assert.Panics(t, func() { f.Fund(ctx, channel.FundingReq{}) }, "Funding with invalid funding req should fail")
	req := channel.FundingReq{
		Params:     &channel.Params{},
		Allocation: &channel.Allocation{},
		Idx:        0,
	}
	assert.NoError(t, f.Fund(ctx, req), "Funding with no assets should succeed")
	parts := []perunwallet.Address{
		&wallet.Address{Address: f.account.Address},
	}
	rng := rand.New(rand.NewSource(1337))
	app := channeltest.NewRandomApp(rng)
	params := channel.NewParamsUnsafe(uint64(0), parts, app.Def(), big.NewInt(rng.Int63()))
	allocation := newValidAllocation(parts, f.ethAssetHolder)
	req = channel.FundingReq{
		Params:     params,
		Allocation: allocation,
		Idx:        0,
	}
	// Test with valid context
	assert.NoError(t, f.Fund(ctx, req), "funding with valid request should succeed")
	assert.NoError(t, f.Fund(ctx, req), "multiple funding should succeed")
	cancel()
	// Test already closed context
	assert.Error(t, f.Fund(ctx, req), "funding with already cancelled context should fail")
}

func TestFunder_Fund_multi(t *testing.T) {
	t.Run("1-party funding", func(t *testing.T) { testFunderFunding(t, 1) })
	t.Run("2-party funding", func(t *testing.T) { testFunderFunding(t, 2) })
	t.Run("3-party funding", func(t *testing.T) { testFunderFunding(t, 3) })
	t.Run("10-party funding", func(t *testing.T) { testFunderFunding(t, 10) })
}

func testFunderFunding(t *testing.T, n int) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	simBackend := test.NewSimulatedBackend()
	// Need unique seed per run.
	seed := time.Now().UnixNano()
	t.Logf("seed is %d", seed)
	rng := rand.New(rand.NewSource(seed))
	ks := ethwallettest.GetKeystore()
	deployAccount := wallettest.NewRandomAccount(rng).(*wallet.Account).Account
	simBackend.FundAddress(ctx, deployAccount.Address)
	contractBackend := NewContractBackend(simBackend, ks, deployAccount)
	// Deploy Assetholder
	assetETH, err := DeployETHAssetholder(ctx, contractBackend, deployAccount.Address)
	if err != nil {
		panic(err)
	}
	t.Logf("asset holder address is %v", assetETH)
	parts := make([]perunwallet.Address, n)
	funders := make([]*Funder, n)
	for i := 0; i < n; i++ {
		acc := wallettest.NewRandomAccount(rng).(*wallet.Account)
		simBackend.FundAddress(ctx, acc.Account.Address)
		parts[i] = acc.Address()
		cb := NewContractBackend(simBackend, ks, acc.Account)
		funders[i] = NewETHFunder(cb, assetETH)
	}
	app := channeltest.NewRandomApp(rng)
	params := channel.NewParamsUnsafe(uint64(0), parts, app.Def(), big.NewInt(rng.Int63()))
	allocation := newValidAllocation(parts, assetETH)
	var wg sync.WaitGroup
	wg.Add(n)
	for i, funder := range funders {
		sleepTime := time.Duration(rng.Int63n(10) + 1)
		go func(i int, funder *Funder) {
			defer wg.Done()
			time.Sleep(sleepTime * time.Millisecond)
			req := channel.FundingReq{
				Params:     params,
				Allocation: allocation,
				Idx:        uint16(i),
			}
			err := funder.Fund(ctx, req)
			assert.NoError(t, err, "funding should succeed")
		}(i, funder)
	}
	wg.Wait()
}

func newSimulatedFunder(t *testing.T) *Funder {
	// Set KeyStore
	wall := new(wallet.Wallet)
	require.NoError(t, wall.Connect(keyDir, password))
	acc := wall.Accounts()[0].(*wallet.Account)
	acc.Unlock(password)
	ks := wall.Ks
	simBackend := test.NewSimulatedBackend()
	simBackend.FundAddress(context.Background(), acc.Account.Address)
	cb := ContractBackend{simBackend, ks, acc.Account}
	// Deploy Assetholder
	assetETH, err := DeployETHAssetholder(context.Background(), cb, acc.Account.Address)
	if err != nil {
		panic(err)
	}
	return NewETHFunder(cb, assetETH)
}

func newValidAllocation(parts []perunwallet.Address, assetETH common.Address) *channel.Allocation {
	// Create assets slice
	assets := []channel.Asset{
		&Asset{Address: assetETH},
	}
	rng := rand.New(rand.NewSource(1337))
	ofparts := make([][]channel.Bal, len(parts))
	for i := 0; i < len(ofparts); i++ {
		ofparts[i] = make([]channel.Bal, len(assets))
		for k := 0; k < len(assets); k++ {
			// create new random balance in range [1,999]
			ofparts[i][k] = big.NewInt(rng.Int63n(999) + 1)
		}
	}
	return &channel.Allocation{
		Assets:  assets,
		OfParts: ofparts,
	}
}
