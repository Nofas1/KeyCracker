package internal

import (
	"strings"

	// "github.com/idsulik/go-collections/queue"
)

type Queue[T any] struct {
	in  []T
	out []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

func (q *Queue[T]) Enqueue(val T) {
	q.in = append(q.in, val)
}

func (q *Queue[T]) Dequeue() (T, bool) {
	if len(q.out) == 0 {
		if len(q.in) == 0 {
			var zero T
			return zero, false
		}
		q.out = make([]T, len(q.in))
		for i, v := range q.in {
			q.out[len(q.in) - 1 - i] = v
		}
		q.in = q.in[:0]
	}
	val := q.out[len(q.out) - 1]
	q.out = q.out[:len(q.out) - 1]
	return val, true
}

func (q *Queue[T]) IsEmpty() bool {
	return len(q.in) == 0 && len(q.out) == 0
}

type ParentData struct {
	Parent string
	Pos int
	Dir int
}

type Node struct {
	state string
	children []*Node
}

type KeyCracker struct{
	relations [][]int
	connections map[string]ParentData
	head *Node
}

func Constructor(rel [][]int, state string) *KeyCracker {
	r := Node{state: state, children: make([]*Node, 0)}
	for i := 0; i < len(rel); i++ {
		rel[i][0]--
		rel[i][1]--
	}
	return &KeyCracker{
		relations: rel,
		connections: make(map[string]ParentData, 0),
		head: &r,
	}
}

func (kc *KeyCracker) BuildTree() {
	q := NewQueue[*Node]()
	kc.connections[kc.head.state] = ParentData{}
	q.Enqueue(kc.head)
	for !q.IsEmpty() {
		f, _ := q.Dequeue()
		if f.state == strings.Repeat("4", len(f.state)) {
			break
		}
		for i := 0; i < len(f.state); i++ {
			left := kc.left(f.state, i)
			right := kc.right(f.state, i)
			if _, ok := kc.connections[left]; !ok {
				kc.connections[left] = ParentData{Parent: f.state, Pos: i, Dir: -1}
				new_node := &Node{state: left}
				f.children = append(f.children, new_node)
				q.Enqueue(new_node)
			}
			if _, ok := kc.connections[right]; !ok {
				kc.connections[right] = ParentData{Parent: f.state, Pos: i, Dir: 1}
				new_node := &Node{state: right}
				f.children = append(f.children, new_node)
				q.Enqueue(new_node)
			}
		}
	}
}

func (kc *KeyCracker) Answer() []ParentData {
	ans := make([]ParentData, 0)
	seek := strings.Repeat("4", len(kc.head.state))
	for len(kc.connections[seek].Parent) != 0 {
		ans = append(ans, kc.connections[seek])
		seek = kc.connections[seek].Parent
	}
	return ans
}

func (kc *KeyCracker) left(state string, pos int) string {
	if state[pos] == '1' {
		return state
	}
	b := []byte(state)
	b[pos]--
	for i := 0; i < len(kc.relations); i++ {
		if kc.relations[i][0] == pos {
			update := int(b[kc.relations[i][1]]) - kc.relations[i][2]
			if update < int('1') || update > int('7') {
				return state
			}
			b[kc.relations[i][1]] = byte(update)
		}
	}
	return string(b)
}

func (kc *KeyCracker) right(state string, pos int) string {
	if state[pos] == '7' {
		return state
	}
	b := []byte(state)
	b[pos]++
	for i := 0; i < len(kc.relations); i++ {
		if kc.relations[i][0] == pos {
			update := int(b[kc.relations[i][1]]) + kc.relations[i][2]
			if update < int('1') || update > int('7') {
				return state
			}
			b[kc.relations[i][1]] = byte(update)
		}
	}
	return string(b)
}
