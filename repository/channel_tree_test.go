package repository

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"testing"
)

var (
	c_a        = uuid.Must(uuid.FromString("6fd36038-dd44-4ac1-bec6-a8a997be6969"))
	c_ab       = uuid.Must(uuid.FromString("390ef0d6-8db2-46c6-afac-4592cab87973"))
	c_abc      = uuid.Must(uuid.FromString("85f7bdb4-ba6b-4bfa-9115-b9dd8fc4c7d1"))
	c_abcd     = uuid.Must(uuid.FromString("84a96808-1fd4-4c5c-aac2-dd8290959270"))
	c_abce     = uuid.Must(uuid.FromString("1907a97e-bb59-4fdf-bf12-a09ef2688be3"))
	c_abf      = uuid.Must(uuid.FromString("0105dd78-5146-4788-8bdb-045d9e4d7ab3"))
	c_abfa     = uuid.Must(uuid.FromString("ee6e2a55-1040-44e4-92c6-e8c1a8d0dd92"))
	c_abb      = uuid.Must(uuid.FromString("c271de50-c0ac-4f06-898f-b1ab09891dd3"))
	c_abbc     = uuid.Must(uuid.FromString("dbb63e83-288f-4ea0-959a-b7235bd17239"))
	c_ad       = uuid.Must(uuid.FromString("25b3a90d-2b75-41d8-9bea-8c1c8b2af166"))
	c_e        = uuid.Must(uuid.FromString("6301b259-2a3e-4a30-9ec8-78737c4b539d"))
	c_ef       = uuid.Must(uuid.FromString("978afa3e-0913-4f52-94d1-c0536a95bb70"))
	c_efg      = uuid.Must(uuid.FromString("fe2477c0-0c44-4247-a556-ab92ac6d5395"))
	c_efgh     = uuid.Must(uuid.FromString("c115b6f5-36f7-428c-9432-580949529acb"))
	c_efghi    = uuid.Must(uuid.FromString("55339fb1-c3c8-4c52-a053-8af0845c45e9"))
	c_efgj     = uuid.Must(uuid.FromString("071eefb9-0a77-4cc4-8d38-5c214aca85f0"))
	c_ek       = uuid.Must(uuid.FromString("2636a1e1-5356-4c5f-8822-c52eeb914689"))
	c_notfound = uuid.Must(uuid.FromString("44bf0189-e3d5-4946-92e7-a196a2a94f98"))
)

/* makeTestChannelTree
a : 6fd36038-dd44-4ac1-bec6-a8a997be6969
├ b : 390ef0d6-8db2-46c6-afac-4592cab87973
│ ├ c : 85f7bdb4-ba6b-4bfa-9115-b9dd8fc4c7d1
│ │ ├ d : 84a96808-1fd4-4c5c-aac2-dd8290959270
│ │ └ e : 1907a97e-bb59-4fdf-bf12-a09ef2688be3
│ ├ f : 0105dd78-5146-4788-8bdb-045d9e4d7ab3
│ │ └ a : ee6e2a55-1040-44e4-92c6-e8c1a8d0dd92
│ └ b : c271de50-c0ac-4f06-898f-b1ab09891dd3
│ 　 └ c : dbb63e83-288f-4ea0-959a-b7235bd17239
└ d : 25b3a90d-2b75-41d8-9bea-8c1c8b2af166
e : 6301b259-2a3e-4a30-9ec8-78737c4b539d
├ f : 978afa3e-0913-4f52-94d1-c0536a95bb70
│ └ g : fe2477c0-0c44-4247-a556-ab92ac6d5395
│ 　 ├ h : c115b6f5-36f7-428c-9432-580949529acb
│ 　 │ └ i : 55339fb1-c3c8-4c52-a053-8af0845c45e9
│ 　 └ j : 071eefb9-0a77-4cc4-8d38-5c214aca85f0
└ k : 2636a1e1-5356-4c5f-8822-c52eeb914689
*/
func makeTestChannelTree(t *testing.T) *channelTreeImpl {
	t.Helper()
	tree, err := makeChannelTree([]*model.Channel{
		{ID: c_a, Name: "a", ParentID: uuid.Nil, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_ab, Name: "b", ParentID: c_a, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_abc, Name: "c", ParentID: c_ab, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_abcd, Name: "d", ParentID: c_abc, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_abce, Name: "e", ParentID: c_abc, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_abf, Name: "f", ParentID: c_ab, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_abfa, Name: "a", ParentID: c_abf, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_abb, Name: "b", ParentID: c_ab, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_abbc, Name: "c", ParentID: c_abb, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_ad, Name: "d", ParentID: c_a, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_e, Name: "e", ParentID: uuid.Nil, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_ef, Name: "f", ParentID: c_e, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_efg, Name: "g", ParentID: c_ef, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_efgh, Name: "h", ParentID: c_efg, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_efghi, Name: "i", ParentID: c_efgh, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_efgj, Name: "j", ParentID: c_efg, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
		{ID: c_ek, Name: "k", ParentID: c_e, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
	})
	assert.NoError(t, err)
	return tree
}

func TestMakeChannelTree(t *testing.T) {
	t.Parallel()
	makeTestChannelTree(t)
}

func TestChannelTreeImpl_GetChildrenIDs(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.ElementsMatch(t, tree.GetChildrenIDs(uuid.Nil), []uuid.UUID{c_a, c_e})
	assert.ElementsMatch(t, tree.GetChildrenIDs(c_a), []uuid.UUID{c_ab, c_ad})
	assert.ElementsMatch(t, tree.GetChildrenIDs(c_ab), []uuid.UUID{c_abc, c_abf, c_abb})
	assert.ElementsMatch(t, tree.GetChildrenIDs(c_abcd), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetChildrenIDs(c_notfound), []uuid.UUID{})
}

func TestChannelTreeImpl_GetDescendantIDs(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.ElementsMatch(t, tree.GetDescendantIDs(uuid.Nil), []uuid.UUID{c_a, c_ab, c_abc, c_abcd, c_abce, c_abf, c_abfa, c_abb, c_abbc, c_ad, c_e, c_ef, c_efg, c_efgh, c_efghi, c_efgj, c_ek})
	assert.ElementsMatch(t, tree.GetDescendantIDs(c_a), []uuid.UUID{c_ab, c_abc, c_abcd, c_abce, c_abf, c_abfa, c_abb, c_abbc, c_ad})
	assert.ElementsMatch(t, tree.GetDescendantIDs(c_ab), []uuid.UUID{c_abc, c_abcd, c_abce, c_abf, c_abfa, c_abb, c_abbc})
	assert.ElementsMatch(t, tree.GetDescendantIDs(c_abcd), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetDescendantIDs(c_notfound), []uuid.UUID{})
}

func TestChannelTreeImpl_GetAscendantIDs(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.ElementsMatch(t, tree.GetAscendantIDs(uuid.Nil), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetAscendantIDs(c_a), []uuid.UUID{})
	assert.ElementsMatch(t, tree.GetAscendantIDs(c_ab), []uuid.UUID{c_a})
	assert.ElementsMatch(t, tree.GetAscendantIDs(c_abcd), []uuid.UUID{c_a, c_ab, c_abc})
	assert.ElementsMatch(t, tree.GetAscendantIDs(c_notfound), []uuid.UUID{})
}

func TestChannelTreeImpl_GetChannelDepth(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.EqualValues(t, 0, tree.GetChannelDepth(uuid.Nil))
	assert.EqualValues(t, 4, tree.GetChannelDepth(c_a))
	assert.EqualValues(t, 3, tree.GetChannelDepth(c_ab))
	assert.EqualValues(t, 1, tree.GetChannelDepth(c_abcd))
	assert.EqualValues(t, 0, tree.GetChannelDepth(c_notfound))
}

func TestChannelTreeImpl_IsChannelPresent(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.False(t, tree.IsChannelPresent(uuid.Nil))
	assert.True(t, tree.IsChannelPresent(c_a))
	assert.True(t, tree.IsChannelPresent(c_ab))
	assert.True(t, tree.IsChannelPresent(c_abcd))
	assert.False(t, tree.IsChannelPresent(c_notfound))
}

func TestChannelTreeImpl_IsChildPresent(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.False(t, tree.IsChildPresent("x", uuid.Nil))
	assert.True(t, tree.IsChildPresent("a", uuid.Nil))
	assert.True(t, tree.IsChildPresent("b", c_a))
	assert.False(t, tree.IsChildPresent("c", c_a))
	assert.True(t, tree.IsChildPresent("c", c_ab))
	assert.False(t, tree.IsChildPresent("a", c_notfound))
}

func TestChannelTreeImpl_GetChannelPath(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.EqualValues(t, "", tree.GetChannelPath(uuid.Nil))
	assert.EqualValues(t, "a", tree.GetChannelPath(c_a))
	assert.EqualValues(t, "a/b", tree.GetChannelPath(c_ab))
	assert.EqualValues(t, "a/b/c/d", tree.GetChannelPath(c_abcd))
	assert.EqualValues(t, "a/b/f/a", tree.GetChannelPath(c_abfa))
	assert.EqualValues(t, "", tree.GetChannelPath(c_notfound))
}

func TestChannelTreeImpl_GetChannelIDFromPath(t *testing.T) {
	t.Parallel()
	tree := makeTestChannelTree(t)

	assert.EqualValues(t, uuid.Nil, tree.GetChannelIDFromPath(""))
	assert.EqualValues(t, c_a, tree.GetChannelIDFromPath("a"))
	assert.EqualValues(t, c_ab, tree.GetChannelIDFromPath("a/b"))
	assert.EqualValues(t, c_abcd, tree.GetChannelIDFromPath("a/b/c/d"))
	assert.EqualValues(t, c_abfa, tree.GetChannelIDFromPath("a/b/f/a"))
	assert.EqualValues(t, uuid.Nil, tree.GetChannelIDFromPath("aaaa"))
}
