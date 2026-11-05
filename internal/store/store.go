// Package store 负责预设文本、分组与标签的本地持久化。
// 所有写操作在单事务内完成，SQLite 以单连接运行并强制外键约束。
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	// ErrDuplicateName 表示分组或标签名称与既有记录冲突（大小写不敏感）。
	ErrDuplicateName = errors.New("名称已存在")
	// ErrInvalidInput 表示写入参数为空。
	ErrInvalidInput = errors.New("输入参数不能为空")
)

// SettingHotkey 是全局唤出快捷键的设置键。
const SettingHotkey = "hotkey"

// SettingOverlayShowUngrouped 是快捷面板是否展示未分组预设文本的设置键。
const SettingOverlayShowUngrouped = "overlay_show_ungrouped"

// Group 是预设文本的单选归属分组，Name 全库唯一。
type Group struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// Tag 是预设文本的多选标签，Name 全库唯一。
type Tag struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// Snippet 是一条预设文本。GroupID 为 nil 表示未分组。
type Snippet struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	GroupID   *int64 `json:"groupId"`
	GroupName string `json:"groupName"`
	Tags      []Tag  `json:"tags"`
	UpdatedAt string `json:"updatedAt"`
}

// SnippetFilter 描述列表查询条件。TagIDs 之间为交集关系。
type SnippetFilter struct {
	Keyword    string
	GroupScope string
	GroupID    *int64
	TagIDs     []int64
}

// SnippetInput 是新建与编辑预设文本的入参。ID 为 0 表示新建。
type SnippetInput struct {
	ID      int64   `json:"id"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	GroupID *int64  `json:"groupId"`
	TagIDs  []int64 `json:"tagIds"`
}

// Store 持有数据库句柄。
type Store struct {
	db *sql.DB
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE COLLATE NOCASE,
		created_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE COLLATE NOCASE,
		created_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS snippets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		group_id INTEGER REFERENCES groups(id) ON DELETE SET NULL,
		updated_at TEXT NOT NULL,
		position INTEGER NOT NULL DEFAULT 0
	)`,
	`CREATE TABLE IF NOT EXISTS snippet_tags (
		snippet_id INTEGER NOT NULL REFERENCES snippets(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (snippet_id, tag_id)
	)`,
	`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
}

// NewStore 打开（必要时创建）数据库并保证表结构就绪。
func NewStore(dbPath string) (*Store, error) {
	if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file:") {
		if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("创建数据库目录: %w", err)
			}
		}
	}

	dsn := dbPath + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	if strings.Contains(dbPath, "?") {
		dsn = dbPath + "&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	// 单连接保证 :memory: 库表可见，并让 DSN 中的外键开关对所有查询生效。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("初始化表结构: %w", err)
		}
	}
	if err := ensureSnippetPosition(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化表结构: %w", err)
	}
	return &Store{db: db}, nil
}

// ensureSnippetPosition adds the manual-order column to a library created
// before it existed, backfilling the old display order in one transaction.
// The column's presence is the migration marker. The pragma rows are closed
// before the transaction starts: the store runs a single connection.
func ensureSnippetPosition(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(snippets)`)
	if err != nil {
		return err
	}
	hasPosition := false
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, typ        string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == "position" {
			hasPosition = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if hasPosition {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`ALTER TABLE snippets ADD COLUMN position INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	ordered, err := tx.Query(`SELECT id FROM snippets ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return err
	}
	var ids []int64
	for ordered.Next() {
		var id int64
		if err := ordered.Scan(&id); err != nil {
			ordered.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := ordered.Err(); err != nil {
		ordered.Close()
		return err
	}
	ordered.Close()
	for position, id := range ids {
		if _, err := tx.Exec(`UPDATE snippets SET position = ? WHERE id = ?`, position, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	return s.db.Close()
}

// GetSetting 读取键值设置；键不存在时返回空串。
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetSetting 写入或覆盖键值设置。
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

// ListGroups 返回全部按名称排序的分组。
func (s *Store) ListGroups() ([]Group, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM groups ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []Group{}
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// SaveGroup 新建（id == 0）或重命名分组。
func (s *Store) SaveGroup(id int64, name string) (Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Group{}, ErrInvalidInput
	}

	var g Group
	if id > 0 {
		res, err := s.db.Exec(`UPDATE groups SET name = ? WHERE id = ?`, name, id)
		if err != nil {
			return Group{}, mapErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return Group{}, sql.ErrNoRows
		}
	} else {
		res, err := s.db.Exec(`INSERT INTO groups (name, created_at) VALUES (?, ?)`, name, now())
		if err != nil {
			return Group{}, mapErr(err)
		}
		id, _ = res.LastInsertId()
	}

	err := s.db.QueryRow(`SELECT id, name, created_at FROM groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &g.CreatedAt)
	if err != nil {
		return Group{}, err
	}
	return g, nil
}

// DeleteGroup 删除分组，其下预设文本在事务内重置为未分组。
func (s *Store) DeleteGroup(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE snippets SET group_id = NULL WHERE group_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM groups WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ListTags 返回全部按名称排序的标签。
func (s *Store) ListTags() ([]Tag, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM tags ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// SaveTag 新建（id == 0）或重命名标签。
func (s *Store) SaveTag(id int64, name string) (Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Tag{}, ErrInvalidInput
	}

	var t Tag
	if id > 0 {
		res, err := s.db.Exec(`UPDATE tags SET name = ? WHERE id = ?`, name, id)
		if err != nil {
			return Tag{}, mapErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return Tag{}, sql.ErrNoRows
		}
	} else {
		res, err := s.db.Exec(`INSERT INTO tags (name, created_at) VALUES (?, ?)`, name, now())
		if err != nil {
			return Tag{}, mapErr(err)
		}
		id, _ = res.LastInsertId()
	}

	err := s.db.QueryRow(`SELECT id, name, created_at FROM tags WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err != nil {
		return Tag{}, err
	}
	return t, nil
}

// DeleteTag 删除标签，关联记录由外键级联清理。
func (s *Store) DeleteTag(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM snippet_tags WHERE tag_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tags WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ListSnippets 按关键字、分组归属与标签交集联合过滤。
func (s *Store) ListSnippets(f SnippetFilter) ([]Snippet, error) {
	var where []string
	var args []any

	if keyword := strings.TrimSpace(f.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		where = append(where, "(s.title LIKE ? OR s.content LIKE ?)")
		args = append(args, like, like)
	}
	switch f.GroupScope {
	case "ungrouped":
		where = append(where, "s.group_id IS NULL")
	case "group":
		if f.GroupID == nil {
			return nil, ErrInvalidInput
		}
		where = append(where, "s.group_id = ?")
		args = append(args, *f.GroupID)
	}

	if tagIDs := uniqueIDs(f.TagIDs); len(tagIDs) > 0 {
		where = append(where, fmt.Sprintf(
			"(SELECT COUNT(DISTINCT st.tag_id) FROM snippet_tags st WHERE st.snippet_id = s.id AND st.tag_id IN (%s)) = ?",
			placeholders(len(tagIDs)),
		))
		for _, id := range tagIDs {
			args = append(args, id)
		}
		args = append(args, len(tagIDs))
	}

	query := `SELECT s.id, s.title, s.content, s.group_id, COALESCE(g.name, ''), s.updated_at
		FROM snippets s LEFT JOIN groups g ON g.id = s.group_id`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY s.position, s.id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	snippets := []Snippet{}
	for rows.Next() {
		var sn Snippet
		var groupID sql.NullInt64
		if err := rows.Scan(&sn.ID, &sn.Title, &sn.Content, &groupID, &sn.GroupName, &sn.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		if groupID.Valid {
			id := groupID.Int64
			sn.GroupID = &id
		}
		snippets = append(snippets, sn)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range snippets {
		tags, err := s.tagsOf(snippets[i].ID)
		if err != nil {
			return nil, err
		}
		snippets[i].Tags = tags
	}
	return snippets, nil
}

// GetSnippetByID 读取单条预设文本的完整内容。
func (s *Store) GetSnippetByID(id int64) (Snippet, error) {
	var sn Snippet
	var groupID sql.NullInt64
	err := s.db.QueryRow(
		`SELECT s.id, s.title, s.content, s.group_id, COALESCE(g.name, ''), s.updated_at
		 FROM snippets s LEFT JOIN groups g ON g.id = s.group_id WHERE s.id = ?`, id,
	).Scan(&sn.ID, &sn.Title, &sn.Content, &groupID, &sn.GroupName, &sn.UpdatedAt)
	if err != nil {
		return Snippet{}, err
	}
	if groupID.Valid {
		gid := groupID.Int64
		sn.GroupID = &gid
	}
	tags, err := s.tagsOf(sn.ID)
	if err != nil {
		return Snippet{}, err
	}
	sn.Tags = tags
	return sn, nil
}

// SaveSnippet 在单事务内写入预设文本并全量替换标签绑定。
func (s *Store) SaveSnippet(in SnippetInput) (Snippet, error) {
	title := strings.TrimSpace(in.Title)
	content := strings.TrimSpace(in.Content)
	if title == "" || content == "" {
		return Snippet{}, ErrInvalidInput
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Snippet{}, err
	}
	defer tx.Rollback()

	id := in.ID
	if id > 0 {
		res, err := tx.Exec(`UPDATE snippets SET title = ?, content = ?, group_id = ?, updated_at = ? WHERE id = ?`,
			title, in.Content, in.GroupID, now(), id)
		if err != nil {
			return Snippet{}, mapErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return Snippet{}, sql.ErrNoRows
		}
	} else {
		// New snippets go above the current top: MIN - 1 keeps the old
		// "newest first" habit without touching anyone else's position.
		var position int64
		if err := tx.QueryRow(`SELECT COALESCE(MIN(position), 0) - 1 FROM snippets`).Scan(&position); err != nil {
			return Snippet{}, err
		}
		res, err := tx.Exec(`INSERT INTO snippets (title, content, group_id, updated_at, position) VALUES (?, ?, ?, ?, ?)`,
			title, in.Content, in.GroupID, now(), position)
		if err != nil {
			return Snippet{}, mapErr(err)
		}
		id, _ = res.LastInsertId()
	}

	if _, err := tx.Exec(`DELETE FROM snippet_tags WHERE snippet_id = ?`, id); err != nil {
		return Snippet{}, err
	}
	for _, tagID := range uniqueIDs(in.TagIDs) {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO snippet_tags (snippet_id, tag_id) VALUES (?, ?)`, id, tagID); err != nil {
			return Snippet{}, mapErr(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Snippet{}, err
	}
	return s.GetSnippetByID(id)
}

// DeleteSnippets 删除多条预设文本，标签关联由外键级联清理。
func (s *Store) DeleteSnippets(ids []int64) error {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt := `DELETE FROM snippets WHERE id IN (` + placeholders(len(ids)) + `)`
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	if _, err := tx.Exec(stmt, args...); err != nil {
		return err
	}
	return tx.Commit()
}

// ReorderSnippets moves ids in front of before (nil: to the end) and
// renumbers every position 0..n-1 in one transaction. Renumbering writes
// the whole table: fine at personal-library scale, and the upgrade path is
// spaced positions if a large library ever makes the write amplification
// matter.
func (s *Store) ReorderSnippets(ids []int64, before *int64) error {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`SELECT id FROM snippets ORDER BY position, id`)
	if err != nil {
		return err
	}
	var all []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		all = append(all, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	moving := make(map[int64]bool, len(ids))
	for _, id := range ids {
		moving[id] = true
	}
	rest := make([]int64, 0, len(all))
	for _, id := range all {
		if !moving[id] {
			rest = append(rest, id)
		}
	}
	at := len(rest)
	if before != nil {
		for i, id := range rest {
			if id == *before {
				at = i
				break
			}
		}
	}

	final := make([]int64, 0, len(all))
	final = append(final, rest[:at]...)
	final = append(final, ids...)
	final = append(final, rest[at:]...)
	for position, id := range final {
		if _, err := tx.Exec(`UPDATE snippets SET position = ? WHERE id = ?`, position, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// BatchMoveGroup 在单事务内将多条预设文本移入同一分组（nil 表示未分组）。
func (s *Store) BatchMoveGroup(snippetIDs []int64, groupID *int64) error {
	snippetIDs = uniqueIDs(snippetIDs)
	if len(snippetIDs) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, id := range snippetIDs {
		if _, err := tx.Exec(`UPDATE snippets SET group_id = ?, updated_at = ? WHERE id = ?`, groupID, now(), id); err != nil {
			return mapErr(err)
		}
	}
	return tx.Commit()
}

// BatchUpdateTags 在单事务内批量绑定与解绑标签。
func (s *Store) BatchUpdateTags(snippetIDs []int64, addTagIDs []int64, removeTagIDs []int64) error {
	snippetIDs = uniqueIDs(snippetIDs)
	if len(snippetIDs) == 0 {
		return nil
	}
	addTagIDs = uniqueIDs(addTagIDs)
	removeTagIDs = uniqueIDs(removeTagIDs)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, snippetID := range snippetIDs {
		for _, tagID := range addTagIDs {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO snippet_tags (snippet_id, tag_id) VALUES (?, ?)`, snippetID, tagID); err != nil {
				return mapErr(err)
			}
		}
		for _, tagID := range removeTagIDs {
			if _, err := tx.Exec(`DELETE FROM snippet_tags WHERE snippet_id = ? AND tag_id = ?`, snippetID, tagID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) tagsOf(snippetID int64) ([]Tag, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, t.created_at FROM tags t
		 JOIN snippet_tags st ON st.tag_id = t.id
		 WHERE st.snippet_id = ? ORDER BY t.name COLLATE NOCASE`, snippetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrDuplicateName
	}
	return err
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func uniqueIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}
