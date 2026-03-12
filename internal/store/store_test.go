package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenClose(t *testing.T) {
	db := testDB(t)
	if db.Path() == "" {
		t.Error("expected non-empty path")
	}
}

func TestFTSEnabled(t *testing.T) {
	db := testDB(t)
	// With sqlite_fts5 build tag, FTS should be enabled.
	if !db.FTSEnabled() {
		t.Log("FTS5 not enabled (build without sqlite_fts5 tag?)")
	}
}

func TestCreatesDirAndFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "nested", "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("db file should exist: %v", err)
	}
}

// --- Chat tests ---

func TestUpsertAndListChats(t *testing.T) {
	db := testDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	chats := []Chat{
		{ChatID: 100, Kind: "dm", Name: "Alice", LastMessageTS: now},
		{ChatID: 200, Kind: "group", Name: "Devs", LastMessageTS: now.Add(-time.Hour)},
		{ChatID: 300, Kind: "channel", Name: "News", LastMessageTS: now.Add(-2 * time.Hour)},
	}
	for _, c := range chats {
		if err := db.UpsertChat(c); err != nil {
			t.Fatalf("upsert chat %d: %v", c.ChatID, err)
		}
	}

	// List should return in order of last_message_ts DESC.
	got, err := db.ListChats(10)
	if err != nil {
		t.Fatalf("list chats: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 chats, got %d", len(got))
	}
	if got[0].ChatID != 100 {
		t.Errorf("expected first chat to be 100, got %d", got[0].ChatID)
	}
}

func TestUpsertChatPreservesNewerTimestamp(t *testing.T) {
	db := testDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "A", LastMessageTS: now}); err != nil {
		t.Fatal(err)
	}
	// Upsert with older timestamp should not downgrade.
	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "A Updated", LastMessageTS: now.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}

	c, err := db.GetChat(1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "A Updated" {
		t.Errorf("expected name 'A Updated', got %q", c.Name)
	}
	if !c.LastMessageTS.Equal(now) {
		t.Errorf("timestamp should not downgrade: got %v, want %v", c.LastMessageTS, now)
	}
}

// --- Contact tests ---

func TestUpsertAndListContacts(t *testing.T) {
	db := testDB(t)

	contacts := []Contact{
		{UserID: 1, FirstName: "Alice", LastName: "Smith", Username: "alice", Phone: "+1111"},
		{UserID: 2, FirstName: "Bob", LastName: "Jones", Username: "bob", Phone: "+2222"},
	}
	for _, c := range contacts {
		if err := db.UpsertContact(c); err != nil {
			t.Fatalf("upsert contact %d: %v", c.UserID, err)
		}
	}

	got, err := db.ListContacts(10)
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(got))
	}
	if got[0].FirstName != "Alice" {
		t.Errorf("expected Alice first, got %q", got[0].FirstName)
	}
}

func TestGetContact(t *testing.T) {
	db := testDB(t)

	if err := db.UpsertContact(Contact{UserID: 42, FirstName: "Test", Username: "testuser"}); err != nil {
		t.Fatal(err)
	}

	c, err := db.GetContact(42)
	if err != nil {
		t.Fatal(err)
	}
	if c.FirstName != "Test" {
		t.Errorf("expected 'Test', got %q", c.FirstName)
	}

	_, err = db.GetContact(999)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

// --- Group tests ---

func TestUpsertAndListGroups(t *testing.T) {
	db := testDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	groups := []Group{
		{ChatID: 100, Title: "Alpha", CreatorID: 1, CreatedTS: now, MemberCount: 5},
		{ChatID: 200, Title: "Beta", CreatorID: 2, CreatedTS: now, MemberCount: 10},
	}
	for _, g := range groups {
		if err := db.UpsertGroup(g); err != nil {
			t.Fatalf("upsert group %d: %v", g.ChatID, err)
		}
	}

	got, err := db.ListGroups(10)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(got))
	}
	if got[0].Title != "Alpha" {
		t.Errorf("expected 'Alpha' first, got %q", got[0].Title)
	}
}

func TestGroupParticipants(t *testing.T) {
	db := testDB(t)

	// Must create group first (FK).
	if err := db.UpsertGroup(Group{ChatID: 100, Title: "Test Group"}); err != nil {
		t.Fatal(err)
	}

	parts := []GroupParticipant{
		{GroupChatID: 100, UserID: 1, Role: "creator"},
		{GroupChatID: 100, UserID: 2, Role: "admin"},
		{GroupChatID: 100, UserID: 3, Role: "member"},
	}
	if err := db.ReplaceGroupParticipants(100, parts); err != nil {
		t.Fatalf("replace participants: %v", err)
	}

	got, err := db.GetGroupParticipants(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(got))
	}

	// Replace with fewer.
	if err := db.ReplaceGroupParticipants(100, parts[:1]); err != nil {
		t.Fatal(err)
	}
	got, err = db.GetGroupParticipants(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 participant after replace, got %d", len(got))
	}
}

// --- Message tests ---

func TestUpsertAndListMessages(t *testing.T) {
	db := testDB(t)

	// Create chat first (FK).
	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	msgs := []UpsertMessageParams{
		{ChatID: 1, MsgID: 1, SenderID: 10, Timestamp: now, FromMe: false, Text: "Hello"},
		{ChatID: 1, MsgID: 2, SenderID: 0, Timestamp: now.Add(time.Second), FromMe: true, Text: "Hi there"},
		{ChatID: 1, MsgID: 3, SenderID: 10, Timestamp: now.Add(2 * time.Second), FromMe: false, Text: "How are you?"},
	}
	for _, m := range msgs {
		if err := db.UpsertMessage(m); err != nil {
			t.Fatalf("upsert msg %d: %v", m.MsgID, err)
		}
	}

	got, err := db.ListMessages(ListMessagesParams{ChatID: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got))
	}
	// Should be ordered by ts DESC.
	if got[0].MsgID != 3 {
		t.Errorf("expected msg 3 first (newest), got %d", got[0].MsgID)
	}
}

func TestUpsertMessageIdempotent(t *testing.T) {
	db := testDB(t)

	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	p := UpsertMessageParams{ChatID: 1, MsgID: 1, SenderID: 10, Timestamp: now, Text: "Original"}

	if err := db.UpsertMessage(p); err != nil {
		t.Fatal(err)
	}

	// Upsert same message with updated text.
	p.Text = "Updated"
	if err := db.UpsertMessage(p); err != nil {
		t.Fatal(err)
	}

	m, err := db.GetMessage(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if m.Text != "Updated" {
		t.Errorf("expected 'Updated', got %q", m.Text)
	}

	// Should still have only one message.
	got, err := db.ListMessages(ListMessagesParams{ChatID: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 message (idempotent), got %d", len(got))
	}
}

func TestGetMessage(t *testing.T) {
	db := testDB(t)

	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	if err := db.UpsertMessage(UpsertMessageParams{ChatID: 1, MsgID: 42, SenderID: 5, Timestamp: now, Text: "test"}); err != nil {
		t.Fatal(err)
	}

	m, err := db.GetMessage(1, 42)
	if err != nil {
		t.Fatal(err)
	}
	if m.Text != "test" || m.SenderID != 5 {
		t.Errorf("unexpected message: %+v", m)
	}

	_, err = db.GetMessage(1, 999)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestListMessagesWithTimeRange(t *testing.T) {
	db := testDB(t)

	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		if err := db.UpsertMessage(UpsertMessageParams{
			ChatID: 1, MsgID: i + 1, Timestamp: ts, Text: "msg",
		}); err != nil {
			t.Fatal(err)
		}
	}

	// After hour 1, before hour 3 → should get msg 2 and 3.
	got, err := db.ListMessages(ListMessagesParams{
		ChatID: 1,
		After:  base.Add(1 * time.Hour),
		Before: base.Add(3 * time.Hour),
		Limit:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 messages in range, got %d", len(got))
	}
}

// --- Search tests ---

func TestSearchMessages(t *testing.T) {
	db := testDB(t)

	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	msgs := []UpsertMessageParams{
		{ChatID: 1, MsgID: 1, Timestamp: now, Text: "Hello world"},
		{ChatID: 1, MsgID: 2, Timestamp: now.Add(time.Second), Text: "Goodbye world"},
		{ChatID: 1, MsgID: 3, Timestamp: now.Add(2 * time.Second), Text: "Nothing here"},
		{ChatID: 1, MsgID: 4, Timestamp: now.Add(3 * time.Second), MediaCaption: "world photo"},
	}
	for _, m := range msgs {
		if err := db.UpsertMessage(m); err != nil {
			t.Fatal(err)
		}
	}

	got, err := db.SearchMessages(SearchMessagesParams{Query: "world", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	// Should find "Hello world", "Goodbye world", and "world photo" (via media_caption).
	if len(got) < 2 {
		t.Fatalf("expected at least 2 results for 'world', got %d", len(got))
	}
}

func TestSearchMessagesNoResults(t *testing.T) {
	db := testDB(t)

	if err := db.UpsertChat(Chat{ChatID: 1, Kind: "dm", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertMessage(UpsertMessageParams{ChatID: 1, MsgID: 1, Text: "hello"}); err != nil {
		t.Fatal(err)
	}

	got, err := db.SearchMessages(SearchMessagesParams{Query: "nonexistent", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 results, got %d", len(got))
	}
}
