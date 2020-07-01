package channel

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"testing"
)

var (
	cA        = uuid.Must(uuid.FromString("6fd36038-dd44-4ac1-bec6-a8a997be6969"))
	cAB       = uuid.Must(uuid.FromString("390ef0d6-8db2-46c6-afac-4592cab87973"))
	cABC      = uuid.Must(uuid.FromString("85f7bdb4-ba6b-4bfa-9115-b9dd8fc4c7d1"))
	cABCD     = uuid.Must(uuid.FromString("84a96808-1fd4-4c5c-aac2-dd8290959270"))
	cABCE     = uuid.Must(uuid.FromString("1907a97e-bb59-4fdf-bf12-a09ef2688be3"))
	cABF      = uuid.Must(uuid.FromString("0105dd78-5146-4788-8bdb-045d9e4d7ab3"))
	cABFA     = uuid.Must(uuid.FromString("ee6e2a55-1040-44e4-92c6-e8c1a8d0dd92"))
	cABB      = uuid.Must(uuid.FromString("c271de50-c0ac-4f06-898f-b1ab09891dd3"))
	cABBC     = uuid.Must(uuid.FromString("dbb63e83-288f-4ea0-959a-b7235bd17239"))
	cAD       = uuid.Must(uuid.FromString("25b3a90d-2b75-41d8-9bea-8c1c8b2af166"))
	cE        = uuid.Must(uuid.FromString("6301b259-2a3e-4a30-9ec8-78737c4b539d"))
	cEF       = uuid.Must(uuid.FromString("978afa3e-0913-4f52-94d1-c0536a95bb70"))
	cEFG      = uuid.Must(uuid.FromString("fe2477c0-0c44-4247-a556-ab92ac6d5395"))
	cEFGH     = uuid.Must(uuid.FromString("c115b6f5-36f7-428c-9432-580949529acb"))
	cEFGHI    = uuid.Must(uuid.FromString("55339fb1-c3c8-4c52-a053-8af0845c45e9"))
	cEFGJ     = uuid.Must(uuid.FromString("071eefb9-0a77-4cc4-8d38-5c214aca85f0"))
	cEK       = uuid.Must(uuid.FromString("2636a1e1-5356-4c5f-8822-c52eeb914689"))
	cNotFound = uuid.Must(uuid.FromString("44bf0189-e3d5-4946-92e7-a196a2a94f98"))
)

/* makeTestChannelTree
a : 6fd36038-dd44-4ac1-bec6-a8a997be6969
├ b : 390ef0d6-8db2-46c6-afac-4592cab87973
│ ├ c : 85f7bdb4-ba6b-4bfa-9115-b9dd8fc4c7d1
│ │ ├ d : 84a96808-1fd4-4c5c-aac2-dd8290959270
│ │ └ e : 1907a97e-bb59-4fdf-bf12-a09ef2688be3
│ ├ f : 0105dd78-5146-4788-8bdb-045d9e4d7ab3
│ │ └ a : ee6e2a55-1040-44e4-92c6-e8c1a8d0dd92
│ └ b : c271de50-c0ac-4f06-898f-b1ab09891dd3 archived
│ 　 └ c : dbb63e83-288f-4ea0-959a-b7235bd17239 archived
└ d : 25b3a90d-2b75-41d8-9bea-8c1c8b2af166
e : 6301b259-2a3e-4a30-9ec8-78737c4b539d force
├ f : 978afa3e-0913-4f52-94d1-c0536a95bb70
│ └ g : fe2477c0-0c44-4247-a556-ab92ac6d5395
│ 　 ├ h : c115b6f5-36f7-428c-9432-580949529acb
│ 　 │ └ i : 55339fb1-c3c8-4c52-a053-8af0845c45e9
│ 　 └ j : 071eefb9-0a77-4cc4-8d38-5c214aca85f0
└ k : 2636a1e1-5356-4c5f-8822-c52eeb914689
*/
func makeTestChannelTree(t *testing.T) *treeImpl {
	t.Helper()
	tree, err := makeChannelTree([]*model.Channel{
		{ID: cA, Name: "a", ParentID: uuid.Nil, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cAB, Name: "b", ParentID: cA, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cABC, Name: "c", ParentID: cAB, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cABCD, Name: "d", ParentID: cABC, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cABCE, Name: "e", ParentID: cABC, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cABF, Name: "f", ParentID: cAB, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cABFA, Name: "a", ParentID: cABF, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cABB, Name: "b", ParentID: cAB, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
		{ID: cABBC, Name: "c", ParentID: cABB, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
		{ID: cAD, Name: "d", ParentID: cA, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cE, Name: "e", ParentID: uuid.Nil, Topic: "", IsForced: true, IsPublic: true, IsVisible: true},
		{ID: cEF, Name: "f", ParentID: cE, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cEFG, Name: "g", ParentID: cEF, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cEFGH, Name: "h", ParentID: cEFG, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cEFGHI, Name: "i", ParentID: cEFGH, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cEFGJ, Name: "j", ParentID: cEFG, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: cEK, Name: "k", ParentID: cE, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
	})
	assert.NoError(t, err)
	return tree
}

func TestMakeChannelTree(t *testing.T) {
	t.Parallel()
	makeTestChannelTree(t)
}

func TestTreeImpl_move(t *testing.T) {
	t.Parallel()
	original := makeTestChannelTree(t)
	tree := makeTestChannelTree(t)

	// (root)/e/kを(root)/kに移動
	tree.move(cEK, optional.UUIDFrom(uuid.Nil), optional.String{})
	assert.Len(t, tree.roots, len(original.roots)+1)
	assert.False(t, tree.isChildPresent("k", cE))
	assert.True(t, tree.isChildPresent("k", uuid.Nil))

	// (root)/kを(root)/e/f/g/kに移動
	tree.move(cEK, optional.UUIDFrom(cEFG), optional.String{})
	assert.Len(t, tree.roots, len(original.roots))
	assert.True(t, tree.isChildPresent("k", cEFG))
	assert.False(t, tree.isChildPresent("k", uuid.Nil))

	// (root)/e/f/g/kを(root)/kに移動
	tree.move(cEK, optional.UUIDFrom(uuid.Nil), optional.String{})
	assert.Len(t, tree.roots, len(original.roots)+1)
	assert.False(t, tree.isChildPresent("k", cEFG))
	assert.True(t, tree.isChildPresent("k", uuid.Nil))

}

func TestChannelTreeImpl_GetChildrenIDs(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.ElementsMatch(t, tree.GetChildrenIDs(uuid.Nil), []uuid.UUID{cA, cE})
	assert.ElementsMatch(t, tree.GetChildrenIDs(cA), []uuid.UUID{cAB, cAD})
	assert.ElementsMatch(t, tree.GetChildrenIDs(cAB), []uuid.UUID{cABC, cABF, cABB})
	assert.ElementsMatch(t, tree.GetChildrenIDs(cABCD), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetChildrenIDs(cNotFound), []uuid.UUID{})
}

func TestChannelTreeImpl_GetDescendantIDs(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.ElementsMatch(t, tree.GetDescendantIDs(uuid.Nil), []uuid.UUID{cA, cAB, cABC, cABCD, cABCE, cABF, cABFA, cABB, cABBC, cAD, cE, cEF, cEFG, cEFGH, cEFGHI, cEFGJ, cEK})
	assert.ElementsMatch(t, tree.GetDescendantIDs(cA), []uuid.UUID{cAB, cABC, cABCD, cABCE, cABF, cABFA, cABB, cABBC, cAD})
	assert.ElementsMatch(t, tree.GetDescendantIDs(cAB), []uuid.UUID{cABC, cABCD, cABCE, cABF, cABFA, cABB, cABBC})
	assert.ElementsMatch(t, tree.GetDescendantIDs(cABCD), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetDescendantIDs(cNotFound), []uuid.UUID{})
}

func TestChannelTreeImpl_GetAscendantIDs(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.ElementsMatch(t, tree.GetAscendantIDs(uuid.Nil), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetAscendantIDs(cA), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetAscendantIDs(cAB), []uuid.UUID{cA})
	assert.ElementsMatch(t, tree.GetAscendantIDs(cABCD), []uuid.UUID{cA, cAB, cABC})
	assert.ElementsMatch(t, tree.GetAscendantIDs(cNotFound), []uuid.UUID{})
}

func TestChannelTreeImpl_GetChannelDepth(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.EqualValues(t, 0, tree.GetChannelDepth(uuid.Nil))
	assert.EqualValues(t, 4, tree.GetChannelDepth(cA))
	assert.EqualValues(t, 3, tree.GetChannelDepth(cAB))
	assert.EqualValues(t, 1, tree.GetChannelDepth(cABCD))
	assert.EqualValues(t, 0, tree.GetChannelDepth(cNotFound))
}

func TestChannelTreeImpl_IsChannelPresent(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.False(t, tree.IsChannelPresent(uuid.Nil))
	assert.True(t, tree.IsChannelPresent(cA))
	assert.True(t, tree.IsChannelPresent(cAB))
	assert.True(t, tree.IsChannelPresent(cABCD))
	assert.False(t, tree.IsChannelPresent(cNotFound))
}

func TestChannelTreeImpl_IsChildPresent(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.False(t, tree.IsChildPresent("x", uuid.Nil))
	assert.True(t, tree.IsChildPresent("a", uuid.Nil))
	assert.True(t, tree.IsChildPresent("b", cA))
	assert.False(t, tree.IsChildPresent("c", cA))
	assert.True(t, tree.IsChildPresent("c", cAB))
	assert.False(t, tree.IsChildPresent("a", cNotFound))
}

func TestChannelTreeImpl_GetChannelPath(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.EqualValues(t, "", tree.GetChannelPath(uuid.Nil))
	assert.EqualValues(t, "a", tree.GetChannelPath(cA))
	assert.EqualValues(t, "a/b", tree.GetChannelPath(cAB))
	assert.EqualValues(t, "a/b/c/d", tree.GetChannelPath(cABCD))
	assert.EqualValues(t, "a/b/f/a", tree.GetChannelPath(cABFA))
	assert.EqualValues(t, "", tree.GetChannelPath(cNotFound))
}

func TestChannelTreeImpl_GetChannelIDFromPath(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.EqualValues(t, uuid.Nil, tree.GetChannelIDFromPath(""))
	assert.EqualValues(t, cA, tree.GetChannelIDFromPath("a"))
	assert.EqualValues(t, cAB, tree.GetChannelIDFromPath("a/b"))
	assert.EqualValues(t, cABCD, tree.GetChannelIDFromPath("a/b/c/d"))
	assert.EqualValues(t, cABFA, tree.GetChannelIDFromPath("a/b/f/a"))
	assert.EqualValues(t, uuid.Nil, tree.GetChannelIDFromPath("aaaa"))
}

func TestChannelTreeImpl_IsForceChannel(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.True(t, tree.IsForceChannel(cE))
	assert.False(t, tree.IsForceChannel(cA))
	assert.False(t, tree.IsForceChannel(uuid.Nil))
}

func TestChannelTreeImpl_IsArchivedChannel(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.True(t, tree.IsArchivedChannel(cABB))
	assert.True(t, tree.IsArchivedChannel(cABBC))
	assert.False(t, tree.IsArchivedChannel(cA))
	assert.False(t, tree.IsArchivedChannel(uuid.Nil))
}
