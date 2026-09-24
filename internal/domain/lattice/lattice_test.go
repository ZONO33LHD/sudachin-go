package lattice

import (
	"errors"
	"slices"
	"testing"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

type costMatrix map[[2]uint16]int16

func (m costMatrix) ConnectCost(left, right uint16) int16 { return m[[2]uint16{left, right}] }

func node(begin, end int, id uint16, cost int16) Node {
	return Node{Begin: begin, End: end, LeftID: id, RightID: id, Cost: cost, WordID: word.ID(id)}
}

func TestBestPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		conn  costMatrix
		nodes []Node
		want  []word.ID
	}{
		{
			name:  "one long word is cheaper than two short words",
			nodes: []Node{node(0, 1, 1, 100), node(1, 2, 2, 100), node(0, 2, 3, 150)},
			want:  []word.ID{3},
		},
		{
			name:  "connection cost changes the answer",
			conn:  costMatrix{{0, 3}: 1000},
			nodes: []Node{node(0, 1, 1, 100), node(1, 2, 2, 100), node(0, 2, 3, 150)},
			want:  []word.ID{1, 2},
		},
		{
			name:  "ties are broken by insertion order",
			nodes: []Node{node(0, 2, 4, 10), node(0, 2, 5, 10)},
			want:  []word.ID{4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := New(2)
			for _, n := range tt.nodes {
				l.Insert(n, tt.conn)
			}
			path, err := l.BestPath(tt.conn)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]word.ID, len(path))
			for i, n := range path {
				got[i] = n.WordID
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("path = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBestPathWithoutReachingEOS(t *testing.T) {
	t.Parallel()
	l := New(3)
	l.Insert(node(0, 1, 1, 0), costMatrix{})
	if _, err := l.BestPath(costMatrix{}); !errors.Is(err, ErrNoPath) {
		t.Errorf("BestPath() error = %v, want ErrNoPath", err)
	}
}
