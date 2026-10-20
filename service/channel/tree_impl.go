package channel

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

type treeImpl struct {
	nodes map[uuid.UUID]*channelNode
	roots map[uuid.UUID]*channelNode
	paths map[uuid.UUID]string
	json  []byte
	sync.RWMutex
}

type channelNode struct {
	id        uuid.UUID                  // 不変
	creatorID uuid.UUID                  // 不変
	createdAt time.Time                  // 不変
	parent    *channelNode               // Treeでロック
	children  map[uuid.UUID]*channelNode // Treeでロック
	name      string                     // Treeでロック
	topic     string                     // Nodeでロック
	archived  bool                       // Nodeでロック
	force     bool                       // Nodeでロック
	updaterID uuid.UUID                  // Nodeでロック
	updatedAt time.Time                  // Nodeでロック
	sync.RWMutex
}

// MarshalJSON implements json.Marshaler interface
func (n *channelNode) MarshalJSON() ([]byte, error) {
	n.RLock()
	defer n.RUnlock()
	v := map[string]interface{}{
		"id":       n.id,
		"name":     n.name,
		"topic":    n.topic,
		"children": n.getChildrenIDs(),
		"archived": n.archived,
		"force":    n.force,
	}
	if n.parent == nil {
		v["parentId"] = nil
	} else {
		v["parentId"] = n.parent.id
	}
	return jsonIter.ConfigFastest.Marshal(v)
}

func (n *channelNode) getChildrenIDs() []uuid.UUID {
	res := make([]uuid.UUID, 0, len(n.children))
	for id := range n.children {
		res = append(res, id)
	}
	return res
}

func (n *channelNode) getChannelDepth() int {
	maxDepth := 0
	for _, c := range n.children {
		d := c.getChannelDepth()
		if maxDepth < d {
			maxDepth = d
		}
	}
	return maxDepth + 1
}

func (n *channelNode) getDescendantIDs() []uuid.UUID {
	var descendants []uuid.UUID
	descendants = append(descendants, n.getChildrenIDs()...)
	for _, c := range n.children {
		descendants = append(descendants, c.getDescendantIDs()...)
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

func (n *channelNode) convertToModel() *model.Channel {
	n.RLock()
	defer n.RUnlock()
	ch := &model.Channel{
		ID:         n.id,
		Name:       n.name,
		Topic:      n.topic,
		IsForced:   n.force,
		IsPublic:   true,
		IsVisible:  !n.archived,
		CreatorID:  n.creatorID,
		UpdaterID:  n.updaterID,
		CreatedAt:  n.createdAt,
		UpdatedAt:  n.updatedAt,
		ChildrenID: n.getChildrenIDs(),
	}
	if n.parent != nil {
		ch.ParentID = n.parent.id
	} else {
		ch.ParentID = pubChannelRootUUID
	}
	return ch
}

func constructChannelNode(chMap map[uuid.UUID]*model.Channel, tree *treeImpl, id uuid.UUID) (*channelNode, error) {
	n, ok := tree.nodes[id]
	if ok {
		return n, nil
	}

	ch, ok := chMap[id]
	if !ok {
		return nil, fmt.Errorf("channel %s was not found", id)
	}

	n = &channelNode{
		id:        ch.ID,
		name:      ch.Name,
		topic:     ch.Topic,
		archived:  ch.IsArchived(),
		force:     ch.IsForced,
		children:  map[uuid.UUID]*channelNode{},
		creatorID: ch.CreatorID,
		updaterID: ch.UpdaterID,
		createdAt: ch.CreatedAt,
		updatedAt: ch.UpdatedAt,
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

func makeChannelTree(channels []*model.Channel) (*treeImpl, error) {
	var (
		chMap = map[uuid.UUID]*model.Channel{}
		ct    = &treeImpl{
			nodes: map[uuid.UUID]*channelNode{},
			roots: map[uuid.UUID]*channelNode{},
			paths: map[uuid.UUID]string{},
		}
	)
	for _, ch := range channels {
		chMap[ch.ID] = ch
	}
	for cid := range chMap {
		n, err := constructChannelNode(chMap, ct, cid)
		if err != nil {
			return nil, err
		}
		if n.parent == nil {
			ct.roots[cid] = n
		}
	}
	ct.regenerateJSON()
	return ct, nil
}

func (ct *treeImpl) add(ch *model.Channel) {
	n := &channelNode{
		id:        ch.ID,
		name:      ch.Name,
		topic:     ch.Topic,
		archived:  ch.IsArchived(),
		force:     ch.IsForced,
		children:  map[uuid.UUID]*channelNode{},
		creatorID: ch.CreatorID,
		updaterID: ch.UpdaterID,
		createdAt: ch.CreatedAt,
		updatedAt: ch.UpdatedAt,
	}
	if ch.ParentID == uuid.Nil {
		// ルート
		ct.roots[n.id] = n
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
	ct.regenerateJSON()
}

func (ct *treeImpl) move(id uuid.UUID, newParent optional.Of[uuid.UUID], newName optional.Of[string]) {
	n, ok := ct.nodes[id]
	if !ok {
		panic("assert !ok = false")
	}

	if newName.Valid {
		n.name = newName.V
	}
	if newParent.Valid {
		if n.parent != nil {
			delete(n.parent.children, n.id)
		} else {
			delete(ct.roots, n.id)
		}
		if newParent.V == uuid.Nil {
			n.parent = nil
			ct.roots[n.id] = n
		} else {
			p, ok := ct.nodes[newParent.V]
			if !ok {
				panic("assert !ok = false")
			}
			n.parent = p
			p.children[n.id] = n
		}
	}
	ct.recalculatePath(n)
	ct.regenerateJSON()
}

func (ct *treeImpl) updateSingle(id uuid.UUID, ch *model.Channel) {
	ct.update(id, ch)
	ct.regenerateJSON()
}

func (ct *treeImpl) updateMultiple(chs []*model.Channel) {
	for _, ch := range chs {
		ct.update(ch.ID, ch)
	}
	ct.regenerateJSON()
}

func (ct *treeImpl) update(id uuid.UUID, ch *model.Channel) {
	n, ok := ct.nodes[id]
	if !ok {
		panic("assert !ok = false")
	}

	n.Lock()
	n.topic = ch.Topic
	n.archived = !ch.IsVisible
	n.force = ch.IsForced
	n.updaterID = ch.UpdaterID
	n.updatedAt = ch.UpdatedAt
	n.Unlock()
}

func (ct *treeImpl) recalculatePath(n *channelNode) {
	if n.parent == nil {
		ct.paths[n.id] = n.name
	} else {
		ct.paths[n.id] = ct.paths[n.parent.id] + "/" + n.name
	}
	for _, c := range n.children {
		ct.recalculatePath(c)
	}
}

func (ct *treeImpl) regenerateJSON() {
	arr := make([]*channelNode, 0, len(ct.nodes))
	for _, node := range ct.nodes {
		arr = append(arr, node)
	}
	b, err := jsonIter.ConfigFastest.Marshal(arr)
	if err != nil {
		panic(err)
	}
	ct.json = b
}

func (ct *treeImpl) GetModel(id uuid.UUID) (*model.Channel, error) {
	ct.RLock()
	defer ct.RUnlock()
	n, ok := ct.nodes[id]
	if !ok {
		return nil, ErrChannelNotFound
	}
	return n.convertToModel(), nil
}

// GetChildrenIDs 子チャンネルのIDの配列を取得する
func (ct *treeImpl) GetChildrenIDs(id uuid.UUID) []uuid.UUID {
	ct.RLock()
	defer ct.RUnlock()
	return ct.getChildrenIDs(id)
}

func (ct *treeImpl) getChildrenIDs(id uuid.UUID) []uuid.UUID {
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
func (ct *treeImpl) GetDescendantIDs(id uuid.UUID) []uuid.UUID {
	ct.RLock()
	defer ct.RUnlock()
	return ct.getDescendantIDs(id)
}

func (ct *treeImpl) getDescendantIDs(id uuid.UUID) []uuid.UUID {
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
func (ct *treeImpl) GetAscendantIDs(id uuid.UUID) []uuid.UUID {
	ct.RLock()
	defer ct.RUnlock()
	return ct.getAscendantIDs(id)
}

func (ct *treeImpl) getAscendantIDs(id uuid.UUID) []uuid.UUID {
	if n, ok := ct.nodes[id]; ok {
		return n.getAscendantIDs()
	}
	return []uuid.UUID{}
}

// GetChannelDepth 指定したチャンネル木の深さを取得する
func (ct *treeImpl) GetChannelDepth(id uuid.UUID) int {
	ct.RLock()
	defer ct.RUnlock()
	return ct.getChannelDepth(id)
}

func (ct *treeImpl) getChannelDepth(id uuid.UUID) int {
	if n, ok := ct.nodes[id]; ok {
		return n.getChannelDepth()
	}
	return 0
}

// IsChildPresent 指定したnameのチャンネルが指定したチャンネルの子に存在するか
func (ct *treeImpl) IsChildPresent(name string, parent uuid.UUID) bool {
	ct.RLock()
	defer ct.RUnlock()
	return ct.isChildPresent(name, parent)
}

func (ct *treeImpl) isChildPresent(name string, parent uuid.UUID) bool {
	name = strings.ToLower(name)
	if parent == uuid.Nil {
		for _, n := range ct.roots {
			if strings.ToLower(n.name) == name {
				return true
			}
		}
		return false
	}
	if n, ok := ct.nodes[parent]; ok {
		for _, n := range n.children {
			if strings.ToLower(n.name) == name {
				return true
			}
		}
	}
	return false
}

// GetChannelPath 指定したチャンネルのパスを取得する
func (ct *treeImpl) GetChannelPath(id uuid.UUID) string {
	ct.RLock()
	defer ct.RUnlock()
	return ct.getChannelPath(id)
}

func (ct *treeImpl) getChannelPath(id uuid.UUID) string {
	return ct.paths[id]
}

// IsChannelPresent 指定したIDのチャンネルが存在するかどうかを取得する
func (ct *treeImpl) IsChannelPresent(id uuid.UUID) bool {
	ct.RLock()
	defer ct.RUnlock()
	return ct.isChannelPresent(id)
}

func (ct *treeImpl) isChannelPresent(id uuid.UUID) bool {
	_, ok := ct.nodes[id]
	return ok
}

// GetChannelIDFromPath チャンネルパスからチャンネルIDを取得する
func (ct *treeImpl) GetChannelIDFromPath(path string) uuid.UUID {
	ct.RLock()
	defer ct.RUnlock()
	return ct.getChannelIDFromPath(path)
}

func (ct *treeImpl) getChannelIDFromPath(path string) uuid.UUID {
	var (
		id       = uuid.Nil
		children = ct.roots
	)
LevelFor:
	for _, name := range strings.Split(strings.ToLower(path), "/") {
		for cid, n := range children {
			if strings.ToLower(n.name) == name {
				id = cid
				children = n.children
				continue LevelFor
			}
		}
		return uuid.Nil
	}
	return id
}

// IsForceChannel 指定したチャンネルが強制通知チャンネルかどうか
func (ct *treeImpl) IsForceChannel(id uuid.UUID) bool {
	ct.RLock()
	defer ct.RUnlock()
	return ct.isForceChannel(id)
}

func (ct *treeImpl) isForceChannel(id uuid.UUID) bool {
	n, ok := ct.nodes[id]
	if !ok {
		return false
	}
	n.RLock()
	defer n.RUnlock()
	return n.force
}

// IsArchivedChannel 指定したチャンネルがアーカイブされているかどうか
func (ct *treeImpl) IsArchivedChannel(id uuid.UUID) bool {
	ct.RLock()
	defer ct.RUnlock()
	return ct.isArchivedChannel(id)
}

func (ct *treeImpl) isArchivedChannel(id uuid.UUID) bool {
	n, ok := ct.nodes[id]
	if !ok {
		return false
	}
	n.RLock()
	defer n.RUnlock()
	return n.archived
}

// MarshalJSON implements json.Marshaler interface
func (ct *treeImpl) MarshalJSON() ([]byte, error) {
	ct.RLock()
	defer ct.RUnlock()
	return ct.json, nil
}
