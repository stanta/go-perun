// Copyright (c) 2019 The Perun Authors. All rights reserved.
// This file is part of go-perun. Use of this source code is governed by a
// MIT-style license that can be found in the LICENSE file.

package peer

import (
	"sync"

	"github.com/pkg/errors"

	"perun.network/go-perun/log"
	wire "perun.network/go-perun/wire/msg"
)

type subscriptions struct {
	mutex sync.RWMutex
	subs  map[wire.Category][]*Receiver
	peer  *Peer
}

// add adds a receiver to the subscriptions.
// If the receiver was already subscribed, panics.
// If the peer is closed, returns an error.
func (s *subscriptions) add(cat wire.Category, r *Receiver) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.peer.isClosed() {
		return errors.New("peer closed")
	}

	for _, rec := range s.subs[cat] {
		if rec == r {
			log.Panic("duplicate peer subscription")
		}
	}

	s.subs[cat] = append(s.subs[cat], r)

	return nil
}

func (s *subscriptions) delete(cat wire.Category, r *Receiver) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subs := s.subs[cat]
	for i, rec := range s.subs[cat] {
		if rec == r {
			subs[i] = subs[len(subs)-1]
			s.subs[cat] = subs[:len(subs)-1]

			return
		}
	}
}

func (s *subscriptions) isEmpty() bool {
	for _, cat := range s.subs {
		if len(cat) != 0 {
			return false
		}
	}
	return true
}

func (s *subscriptions) put(m wire.Msg, p *Peer) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, rec := range s.subs[m.Category()] {
		rec.msgs <- MsgTuple{p, m}
	}
}

func makeSubscriptions(p *Peer) subscriptions {
	return subscriptions{
		peer: p,
		subs: make(map[wire.Category][]*Receiver),
	}
}
