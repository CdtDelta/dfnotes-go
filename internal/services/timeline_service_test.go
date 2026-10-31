package services

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dfnotes-go/internal/config"
	"dfnotes-go/internal/crypto"
	"dfnotes-go/internal/database"
	"dfnotes-go/internal/models"
	"dfnotes-go/internal/timer"
)

const timelineTestCaseID = "timeline-case-1"

// setupTimelineServiceTest wires a real DB with one user and one case. The
// returned pointer controls the timeline_allow_offsets setting for the service.
func setupTimelineServiceTest(t *testing.T) (*TimelineService, *bool) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pub, priv, err := crypto.GenerateSigningKeyPair()
	if err != nil {
		t.Fatalf("GenerateSigningKeyPair: %v", err)
	}
	salt, _ := crypto.GenerateSalt()
	now := time.Now().UTC().Truncate(time.Second)
	user := &models.UserIdentity{
		UserID:              "timeline-user-1",
		Name:                "Test Examiner",
		Organization:        "Test Org",
		PublicKey:           pub,
		EncryptedPrivateKey: []byte("unused"),
		Salt:                salt,
		CreatedAt:           now,
	}
	if err := database.NewUserRepo(db).Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	c := &models.Case{
		CaseID:         timelineTestCaseID,
		CaseNumber:     "CASE-TIMELINE-001",
		Title:          "Timeline Case",
		Classification: models.ClassificationUnclassified,
		Salt:           salt,
		EncryptedKey:   []byte("unused"),
		CreatedBy:      user.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := database.NewCaseRepo(db).Create(context.Background(), c); err != nil {
		t.Fatalf("create case: %v", err)
	}

	sess := NewSession()
	sess.SetAuthenticated(user, make([]byte, 32), priv)
	allowOffsets := new(bool)
	svc := NewTimelineService(database.NewTimelineRepo(db), sess, timer.New(&config.Config{}, func(int) {}),
		func() bool { return *allowOffsets })
	return svc, allowOffsets
}

func createTimelineEntry(t *testing.T, svc *TimelineService, timestamp string) *models.TimelineEntry {
	t.Helper()
	entry, err := svc.CreateTimelineEntry(context.Background(), models.CreateTimelineEntryRequest{
		CaseID:           timelineTestCaseID,
		Timestamp:        timestamp,
		EventDescription: "event",
	})
	if err != nil {
		t.Fatalf("CreateTimelineEntry(%q): %v", timestamp, err)
	}
	return entry
}

// storedEntry reads through the repo so tests check what persisted, not what
// the service returned.
func storedEntry(t *testing.T, svc *TimelineService, entryID string) *models.TimelineEntry {
	t.Helper()
	entry, err := svc.timelineRepo.GetByID(context.Background(), entryID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	return entry
}

func updateTimestamp(t *testing.T, svc *TimelineService, entryID, timestamp, description string) {
	t.Helper()
	_, err := svc.UpdateTimelineEntry(context.Background(), models.UpdateTimelineEntryRequest{
		EntryID:          entryID,
		Timestamp:        timestamp,
		EventDescription: description,
	})
	if err != nil {
		t.Fatalf("UpdateTimelineEntry(%q): %v", timestamp, err)
	}
}

func TestCreateTimelineEntryOffsetDisallowed(t *testing.T) {
	svc, _ := setupTimelineServiceTest(t)
	_, err := svc.CreateTimelineEntry(context.Background(), models.CreateTimelineEntryRequest{
		CaseID:           timelineTestCaseID,
		Timestamp:        "2026-10-30T10:00:00-05:00",
		EventDescription: "event",
	})
	if err == nil || !strings.Contains(err.Error(), "must be in UTC") {
		t.Fatalf("error = %v, want UTC required", err)
	}
}

func TestCreateTimelineEntryOffsetAllowed(t *testing.T) {
	svc, allowOffsets := setupTimelineServiceTest(t)
	*allowOffsets = true

	created := createTimelineEntry(t, svc, "2026-10-30T10:00:00-05:00")
	got := storedEntry(t, svc, created.EntryID)
	if got.Timestamp != "2026-10-30T15:00:00Z" {
		t.Errorf("timestamp = %q, want canonical UTC", got.Timestamp)
	}
	if got.TimestampAsEntered == nil || *got.TimestampAsEntered != "2026-10-30T10:00:00-05:00" {
		t.Errorf("timestamp_as_entered = %v, want the original input", got.TimestampAsEntered)
	}
}

func TestCreateTimelineEntryCanonicalizesFraction(t *testing.T) {
	svc, _ := setupTimelineServiceTest(t)
	created := createTimelineEntry(t, svc, "2026-10-30T15:00:00.500Z")
	got := storedEntry(t, svc, created.EntryID)
	if got.Timestamp != "2026-10-30T15:00:00.5Z" {
		t.Errorf("timestamp = %q, want trailing zeros trimmed", got.Timestamp)
	}
	if got.TimestampAsEntered != nil {
		t.Errorf("timestamp_as_entered = %q, want nil for Z input", *got.TimestampAsEntered)
	}
}

func TestUpdateTimelineEntryUnchangedTimestampKeepsAsEntered(t *testing.T) {
	svc, allowOffsets := setupTimelineServiceTest(t)
	*allowOffsets = true
	created := createTimelineEntry(t, svc, "2026-10-30T10:00:00-05:00")

	// The kept value must survive the setting being turned off afterwards.
	*allowOffsets = false
	updateTimestamp(t, svc, created.EntryID, "2026-10-30T15:00:00Z", "edited")

	got := storedEntry(t, svc, created.EntryID)
	if got.EventDescription != "edited" {
		t.Errorf("event_description = %q, want edited", got.EventDescription)
	}
	if got.TimestampAsEntered == nil || *got.TimestampAsEntered != "2026-10-30T10:00:00-05:00" {
		t.Errorf("timestamp_as_entered = %v, want it kept", got.TimestampAsEntered)
	}
}

func TestUpdateTimelineEntryNewZTimestampClearsAsEntered(t *testing.T) {
	svc, allowOffsets := setupTimelineServiceTest(t)
	*allowOffsets = true
	created := createTimelineEntry(t, svc, "2026-10-30T10:00:00-05:00")

	updateTimestamp(t, svc, created.EntryID, "2026-10-30T16:00:00Z", "event")

	got := storedEntry(t, svc, created.EntryID)
	if got.Timestamp != "2026-10-30T16:00:00Z" {
		t.Errorf("timestamp = %q, want the new value", got.Timestamp)
	}
	if got.TimestampAsEntered != nil {
		t.Errorf("timestamp_as_entered = %q, want cleared", *got.TimestampAsEntered)
	}
}

func TestUpdateTimelineEntryNewOffsetReplacesAsEntered(t *testing.T) {
	svc, allowOffsets := setupTimelineServiceTest(t)
	*allowOffsets = true
	created := createTimelineEntry(t, svc, "2026-10-30T10:00:00-05:00")

	updateTimestamp(t, svc, created.EntryID, "2026-10-30T12:00:00-04:00", "event")

	got := storedEntry(t, svc, created.EntryID)
	if got.Timestamp != "2026-10-30T16:00:00Z" {
		t.Errorf("timestamp = %q, want converted UTC", got.Timestamp)
	}
	if got.TimestampAsEntered == nil || *got.TimestampAsEntered != "2026-10-30T12:00:00-04:00" {
		t.Errorf("timestamp_as_entered = %v, want the new offset input", got.TimestampAsEntered)
	}
}

func TestGetTimelineEntriesChronological(t *testing.T) {
	svc, _ := setupTimelineServiceTest(t)
	for _, ts := range []string{
		"2026-10-30T15:00:05.5Z",
		"2026-10-30T15:00:05Z",
		"2026-10-30T15:00:04.999999999Z",
	} {
		createTimelineEntry(t, svc, ts)
	}

	entries, err := svc.GetTimelineEntries(context.Background(), timelineTestCaseID)
	if err != nil {
		t.Fatalf("GetTimelineEntries: %v", err)
	}
	want := []string{"2026-10-30T15:00:04.999999999Z", "2026-10-30T15:00:05Z", "2026-10-30T15:00:05.5Z"}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i, e := range entries {
		if e.Timestamp != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, e.Timestamp, want[i])
		}
	}
}
