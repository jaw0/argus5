// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-08 20:29 (EDT)
// Function: queue notifications and send

package notify

import (
	"argus/clock"
)

type queuedat struct {
	// method
	// qtime
	dst   string
	addr  string
	meth  *Method
	lastt int64
	notif []*N
}

// package lock should be already held
func (n *N) maybeQueue() {

	now := clock.Unix()

	n.lock.Lock()
	defer n.lock.Unlock()

	if !n.p.IsActive {
		return
	}

	// initial send/escalate
	if len(n.p.SendTo) > n.p.StepNo {
		s := n.p.SendTo[n.p.StepNo]
		if s.When+n.p.Created <= now {
			addToQueue(n, s.Dst)
			if n.p.StepNo > 0 {
				n.p.Escalated = true
			}
			n.p.StepNo++
		}
	}

	// resend
	if n.cf.Renotify == 0 {
		return
	}

	for i, s := range n.p.SendTo {
		if i >= n.p.StepNo {
			break
		}

		if s.When+n.p.Created+n.cf.Renotify <= now {
			addToQueue(n, s.Dst)
		}
	}
}

// called with package+notify locks held
func addToQueue(n *N, dst []string) {

	for _, d := range dst {
		qd, ok := dstQueue[d]
		if !ok {
			meth, addr := methodForDst(d)
			if meth == nil {
				dl.Problem("cannot determine method to send to '%s'", d)
				n.log(d, "failed")
				continue
			}

			qd = &queuedat{dst: d, meth: meth, addr: addr}
			dstQueue[d] = qd
		}
		qd.notif = append(qd.notif, n)
		n.log(d, "queued")
		n.p.Status[d] = "queued"
		dl.Debug("queued id=%d to %s", n.p.IdNo, d)
	}
}

func runQueues() {

	now := clock.Unix()
	lock.Lock()
	defer lock.Unlock()

	for dst, qd := range dstQueue {
		if len(qd.notif) == 0 {
			// nothing queued for this dst
			continue
		}
		if qd.lastt+qd.meth.Qtime <= now {
			// transmit
			ns := []*N{}

			for _, n := range qd.notif {
				// discard anything no longer active
				if n.p.IsActive {
					ns = append(ns, n)
				}
			}

			if len(ns) == 0 {
				return
			}

			qd.notif = nil
			qd.lastt = now

			if qd.meth.Qtime == 0 {
				// qtime 0 => send one-by-one
				for i, _ := range ns {
					qd.meth.transmit(dst, qd.addr, ns[i:i+1])
				}
			} else {
				qd.meth.transmit(dst, qd.addr, ns)
			}
			for _, n := range ns {
				n.lock.Lock()
				n.p.LastSent = now
				n.log(dst, "transmit")
				n.p.Status[dst] = "sent"
				n.lock.Unlock()

				n.maybeAutoAck()
			}
		}
	}
}
