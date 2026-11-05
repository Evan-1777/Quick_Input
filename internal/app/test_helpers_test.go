package app

import (
	"errors"
	"testing"

	"quick-input/internal/store"
)

type fakePaster struct {
	target int
	texts  []string
}

func (f *fakePaster) CaptureTarget() { f.target++ }
func (f *fakePaster) Paste(text string) error {
	f.texts = append(f.texts, text)
	return nil
}

type fakeShortcuts struct {
	registered map[string]func()
	calls      []string
	fail       map[string]bool
}

func newFakeShortcuts() *fakeShortcuts {
	return &fakeShortcuts{registered: map[string]func(){}, fail: map[string]bool{}}
}

func (f *fakeShortcuts) Register(spec string, fn func()) error {
	if f.fail[spec] {
		return errors.New("taken")
	}
	f.registered[spec] = fn
	f.calls = append(f.calls, spec)
	return nil
}

func (f *fakeShortcuts) Unregister(spec string) {
	delete(f.registered, spec)
	f.calls = append(f.calls, "-"+spec)
}

type fixture struct {
	app     *App
	paster  *fakePaster
	keys    *fakeShortcuts
	groupA  int64
	groupB  int64
	tagA    int64
	tagB    int64
	tagC    int64
	snippet [4]int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	st, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	newGroup := func(name string) int64 {
		g, err := st.SaveGroup(0, name)
		if err != nil {
			t.Fatalf("SaveGroup(%q): %v", name, err)
		}
		return g.ID
	}
	newTag := func(name string) int64 {
		tag, err := st.SaveTag(0, name)
		if err != nil {
			t.Fatalf("SaveTag(%q): %v", name, err)
		}
		return tag.ID
	}
	gA, gB := newGroup("工作"), newGroup("生活")
	tA, tB, tC := newTag("urgent"), newTag("work"), newTag("later")

	newSnippet := func(title, content string, group *int64, tags ...int64) int64 {
		sn, err := st.SaveSnippet(store.SnippetInput{Title: title, Content: content, GroupID: group, TagIDs: tags})
		if err != nil {
			t.Fatalf("SaveSnippet(%q): %v", title, err)
		}
		return sn.ID
	}
	s1 := newSnippet("Greeting", "Hello world", &gA, tA)
	s2 := newSnippet("Farewell", "Goodbye world", &gB, tA, tB)
	s3 := newSnippet("Ungrouped", "hello there", nil, tB)
	s4 := newSnippet("Mixed", "HELLO", &gA, tA, tB)

	paster := &fakePaster{}
	keys := newFakeShortcuts()
	a := New(st, paster, keys)
	a.SetStartupHotkey("Alt+Space", "Alt+Space")
	if err := a.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	return &fixture{
		app: a, paster: paster, keys: keys,
		groupA: gA, groupB: gB, tagA: tA, tagB: tB, tagC: tC,
		snippet: [4]int64{s1, s2, s3, s4},
	}
}
