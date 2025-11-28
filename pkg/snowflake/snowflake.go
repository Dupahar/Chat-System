package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	nodeBits        = 10
	stepBits        = 12
	nodeMax         = -1 ^ (-1 << nodeBits)
	stepMask        = -1 ^ (-1 << stepBits)
	timeShift       = nodeBits + stepBits
	nodeShift       = stepBits
	epoch     int64 = 1704067200000 // 2024-01-01 00:00:00 UTC
)

type Node struct {
	mu    sync.Mutex
	time  int64
	node  int64
	step  int64
	epoch int64
}

func NewNode(node int64) (*Node, error) {
	if node < 0 || node > nodeMax {
		return nil, errors.New("node number must be between 0 and 1023")
	}
	return &Node{
		time:  0,
		node:  node,
		step:  0,
		epoch: epoch,
	}, nil
}

func (n *Node) Generate() int64 {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < n.time {
		// Clock moved backwards, refuse to generate id
		now = n.time
	}

	if n.time == now {
		n.step = (n.step + 1) & stepMask
		if n.step == 0 {
			for now <= n.time {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		n.step = 0
	}

	n.time = now

	return ((now - n.epoch) << timeShift) | (n.node << nodeShift) | n.step
}
