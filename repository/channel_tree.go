package repository

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
	"strings"
	"sync"
)

type channelNode struct {
	id       uuid.UUID                  // 不変なため非ロック
	parent   *channelNode               // Treeでロック
	children map[uuid.UUID]*channelNode // Treeでロック
	name     string                     // Treeでロック
	topic    string                     // Nodeでロック
	archived bool                       // Nodeでロック
	force    bool                       // Nodeでロック
	sync.RWMutex
}

func (n *channelNode) getChildrenIDs() []uuid.UUID {
	res := make([]uuid.UUID, 0, len(n.children))
	for id := range n.children {
		res = append(res, id)
	}
	return res
}

func (n *channelNode) getChannelDepth() int {
	max := 0
	for _, c := range n.children {
		d := c.getChannelDepth()
		if max < d {
			max = d
		}
	}
	return max + 1
}

func (n *channelNode) getDescendantIDs() []uuid.UUID {
	var descendants []uuid.UUID
	descendants = append(descendants, n.getChildrenIDs()...)
	for _, c := range n.children {
		descendants = append(descendants, c.getChildrenIDs()...)
	}
	return descendants
}

func (n *channelNode) getAscendantIDs() []uuid.UUID {
	if n.parent == nil {
		return []uuid.UUID{}
	}
	var ascendants []uuid.UUID
	ascendants = append(ascendants, n.parent.id)
	ascendants = append(ascendants, n.parent.getAscendantIDs()...)
	return ascendants
}

func constructChannelNode(chMap map[uuid.UUID]*model.Channel, tree *channelTree, id uuid.UUID) (*channelNode, error) {
	n, ok := tree.nodes[id]
	if ok {
		return n, nil
	}

	ch, ok := chMap[id]
	if !ok {
		return nil, fmt.Errorf("channel %s was not found", id)
	}

	n = &channelNode{
		id:       ch.ID,
		name:     ch.Name,
		topic:    ch.Topic,
		archived: ch.IsArchived(),
		force:    ch.IsForced,
		children: map[uuid.UUID]*channelNode{},
	}
	if ch.ParentID != uuid.Nil {
		p, err := constructChannelNode(chMap, tree, ch.ParentID)
		if err != nil {
			return nil, fmt.Errorf("inconsistent channel tree: the parent of %s was not found (%w)", n.id, err)
		}
		n.parent = p
		p.children[n.id] = n
		tree.paths[n.id] = tree.paths[p.id] + "/" + n.name
	} else {
		tree.paths[n.id] = n.name
	}

	tree.nodes[n.id] = n
	return n, nil
}

type channelTree struct {
	nodes map[uuid.UUID]*channelNode
	roots map[uuid.UUID]*channelNode
	paths map[uuid.UUID]string
	mu    sync.RWMutex
}

func makeChannelTree(channels []*model.Channel) (*channelTree, error) {
	var (
		roots []uuid.UUID
		chMap = map[uuid.UUID]*model.Channel{}
		ct    = &channelTree{
			nodes: map[uuid.UUID]*channelNode{},
			roots: map[uuid.UUID]*channelNode{},
			paths: map[uuid.UUID]string{},
		}
	)
	for _, ch := range channels {
		chMap[ch.ID] = ch
		if ch.ParentID == uuid.Nil {
			roots = append(roots, ch.ID)
		}
	}
	for _, cid := range roots {
		n, err := constructChannelNode(chMap, ct, cid)
		if err != nil {
			return nil, err
		}
		ct.roots[cid] = n
	}
	return ct, nil
}

func (ct *channelTree) add(ch *model.Channel) {
	n := &channelNode{
		id:       ch.ID,
		name:     ch.Name,
		topic:    ch.Topic,
		archived: ch.IsArchived(),
		force:    ch.IsForced,
		children: map[uuid.UUID]*channelNode{},
	}
	if ch.ParentID == uuid.Nil {
		// ルート
		ct.paths[n.id] = n.name
	} else {
		p, ok := ct.nodes[ch.ParentID]
		if !ok {
			panic("assert !ok = false")
		}
		n.parent = p
		p.children[n.id] = n
		ct.paths[n.id] = ct.paths[p.id] + "/" + n.name
	}
	ct.nodes[n.id] = n
}

func (ct *channelTree) move(id uuid.UUID, newParent uuid.NullUUID, newName null.String) {
	n, ok := ct.nodes[id]
	if !ok {
		panic("assert !ok = false")
	}

	if newName.Valid {
		n.name = newName.String
	}
	if newParent.Valid {
		if n.parent != nil {
			delete(n.parent.children, n.id)
		}
		p, ok := ct.nodes[newParent.UUID]
		if !ok {
			panic("assert !ok = false")
		}
		n.parent = p
		p.children[n.id] = n
	}
	ct.recalculatePath(n)
}

func (ct *channelTree) update(id uuid.UUID, topic null.String, archived null.Bool, force null.Bool) {
	n, ok := ct.nodes[id]
	if !ok {
		panic("assert !ok = false")
	}

	n.Lock()
	defer n.Unlock()
	if topic.Valid {
		n.topic = topic.String
	}
	if archived.Valid {
		n.archived = archived.Bool
	}
	if force.Valid {
		n.force = force.Bool
	}
}

func (ct *channelTree) recalculatePath(n *channelNode) {
	if n.parent == nil {
		ct.paths[n.id] = n.name
	} else {
		ct.paths[n.id] = ct.paths[n.parent.id] + "/" + n.name
	}
	for _, c := range n.children {
		ct.recalculatePath(c)
	}
}

// GetChildrenIDs 子チャンネルのIDの配列を取得する
func (ct *channelTree) GetChildrenIDs(id uuid.UUID) []uuid.UUID {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.getChildrenIDs(id)
}

func (ct *channelTree) getChildrenIDs(id uuid.UUID) []uuid.UUID {
	if id == uuid.Nil {
		var res []uuid.UUID
		for cid := range ct.roots {
			res = append(res, cid)
		}
		return res
	}
	if n, ok := ct.nodes[id]; ok {
		return n.getChildrenIDs()
	}
	return []uuid.UUID{}
}

// GetDescendantIDs 子孫チャンネルのIDの配列を取得する
func (ct *channelTree) GetDescendantIDs(id uuid.UUID) []uuid.UUID {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.getDescendantIDs(id)
}

func (ct *channelTree) getDescendantIDs(id uuid.UUID) []uuid.UUID {
	if id == uuid.Nil {
		var res []uuid.UUID
		for cid, c := range ct.roots {
			res = append(res, cid)
			res = append(res, c.getDescendantIDs()...)
		}
		return res
	}
	if n, ok := ct.nodes[id]; ok {
		return n.getDescendantIDs()
	}
	return []uuid.UUID{}
}

// GetAscendantIDs 祖先チャンネルのIDの配列を取得する
func (ct *channelTree) GetAscendantIDs(id uuid.UUID) []uuid.UUID {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.getAscendantIDs(id)
}

func (ct *channelTree) getAscendantIDs(id uuid.UUID) []uuid.UUID {
	if n, ok := ct.nodes[id]; ok {
		return n.getAscendantIDs()
	}
	return []uuid.UUID{}
}

// GetChannelDepth 指定したチャンネル木の深さを取得する。自分自身は深さに含まれません。
func (ct *channelTree) GetChannelDepth(id uuid.UUID) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.getChannelDepth(id)
}

func (ct *channelTree) getChannelDepth(id uuid.UUID) int {
	if n, ok := ct.nodes[id]; ok {
		return n.getChannelDepth()
	}
	return 0
}

// IsChildPresent 指定したnameのチャンネルが指定したチャンネルの子に存在するか
func (ct *channelTree) IsChildPresent(name string, parent uuid.UUID) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.isChildPresent(name, parent)
}

func (ct *channelTree) isChildPresent(name string, parent uuid.UUID) bool {
	if parent == uuid.Nil {
		for _, n := range ct.roots {
			if n.name == name {
				return true
			}
		}
	}
	if n, ok := ct.nodes[parent]; ok {
		for _, n := range n.children {
			if n.name == name {
				return true
			}
		}
	}
	return false
}

// GetChannelPath 指定したチャンネルのパスを取得する
func (ct *channelTree) GetChannelPath(id uuid.UUID) string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.getChannelPath(id)
}

func (ct *channelTree) getChannelPath(id uuid.UUID) string {
	return ct.paths[id]
}

// IsChannelPresent 指定したIDのチャンネルが存在するかどうかを取得する
func (ct *channelTree) IsChannelPresent(id uuid.UUID) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.isChannelPresent(id)
}

func (ct *channelTree) isChannelPresent(id uuid.UUID) bool {
	_, ok := ct.nodes[id]
	return ok
}

// GetChannelIDFromPath チャンネルパスからチャンネルIDを取得する
func (ct *channelTree) GetChannelIDFromPath(path string) uuid.UUID {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.getChannelIDFromPath(path)
}

func (ct *channelTree) getChannelIDFromPath(path string) uuid.UUID {
	var (
		id       = uuid.Nil
		children = ct.roots
	)
	for _, name := range strings.Split(path, "/") {
		for cid, n := range children {
			if n.name == name {
				id = cid
				children = n.children
				break
			}
		}
	}
	return id
}
