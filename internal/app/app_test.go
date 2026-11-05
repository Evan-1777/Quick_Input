package app

import (
	"testing"

	"quick-input/internal/store"
)

func TestShowUngroupedSetting(t *testing.T) {
	st, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer st.Close()

	a := New(st, &fakePaster{}, newFakeShortcuts())
	if a.showUngrouped {
		t.Fatal("默认应隐藏未分组")
	}
	if err := a.SetShowUngrouped(true); err != nil {
		t.Fatalf("开启显示未分组: %v", err)
	}
	if !a.showUngrouped {
		t.Fatal("开启后字段应为 true")
	}
	if reopened := New(st, &fakePaster{}, newFakeShortcuts()); !reopened.showUngrouped {
		t.Fatal("设置未跨应用实例生效")
	}

	a.overlay.filter = FilterState{Scope: scopeUngrouped, Keyword: "x", TagIDs: []int64{7}}
	if err := a.SetShowUngrouped(false); err != nil {
		t.Fatalf("关闭显示未分组: %v", err)
	}
	if a.showUngrouped || a.overlay.filter.Scope != scopeAll ||
		a.overlay.filter.Keyword != "" || a.overlay.filter.TagIDs != nil {
		t.Fatalf("关闭后筛选未重置: %+v", a.overlay.filter)
	}

	if err := st.Close(); err != nil {
		t.Fatalf("关闭库: %v", err)
	}
	if err := a.SetShowUngrouped(true); err == nil {
		t.Fatal("写库失败应返回错误")
	}
	if a.showUngrouped {
		t.Fatal("写库失败不应改变内存值")
	}
}
