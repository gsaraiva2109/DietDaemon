package store

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestGetPhotosData_BatchReturnsCorrectData(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	uid := "u-photos"
	mustUser(t, s, types.User{ID: uid})

	want := map[string][]byte{
		"photo1": []byte("front-bytes"),
		"photo2": []byte("side-bytes"),
		"photo3": []byte("back-bytes"),
	}
	for id, data := range want {
		if err := s.UploadPhoto(ctx(), types.ProgressPhoto{
			ID: id, UserID: uid, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: data,
		}); err != nil {
			t.Fatalf("UploadPhoto(%s): %v", id, err)
		}
	}

	got, err := s.GetPhotosData(ctx(), uid, []string{"photo1", "photo2", "photo3"})
	if err != nil {
		t.Fatalf("GetPhotosData: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("GetPhotosData returned %d entries, want %d", len(got), len(want))
	}
	for id, data := range want {
		if string(got[id]) != string(data) {
			t.Fatalf("GetPhotosData[%s] = %q, want %q", id, got[id], data)
		}
	}
}

func TestGetPhotosData_EmptyInput(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	got, err := s.GetPhotosData(ctx(), "any-user", nil)
	if err != nil {
		t.Fatalf("GetPhotosData(nil): %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetPhotosData(nil) = %v, want empty map", got)
	}
}

func TestGetPhotosData_MissingIDIgnored(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	uid := "u-photos-missing"
	mustUser(t, s, types.User{ID: uid})
	if err := s.UploadPhoto(ctx(), types.ProgressPhoto{
		ID: "photo1", UserID: uid, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("bytes"),
	}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}

	got, err := s.GetPhotosData(ctx(), uid, []string{"photo1", "no-such-id"})
	if err != nil {
		t.Fatalf("GetPhotosData: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("GetPhotosData returned %d entries, want 1 (missing id silently absent)", len(got))
	}
	if string(got["photo1"]) != "bytes" {
		t.Fatalf("GetPhotosData[photo1] = %q, want %q", got["photo1"], "bytes")
	}
	if _, ok := got["no-such-id"]; ok {
		t.Fatalf("expected no-such-id absent from result map")
	}
}

// TestGetPhotoData_CrossUserAccessDenied pins the security fix: the SQL
// query itself is scoped by user_id, so a photoID owned by a different user
// returns ErrNotFound at the store layer — not just a handler-level check.
func TestGetPhotoData_CrossUserAccessDenied(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	owner := "u-owner"
	attacker := "u-attacker"
	mustUser(t, s, types.User{ID: owner})
	mustUser(t, s, types.User{ID: attacker})

	if err := s.UploadPhoto(ctx(), types.ProgressPhoto{
		ID: "owner-photo", UserID: owner, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("secret"),
	}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}

	// Owner can fetch their own photo.
	got, err := s.GetPhotoData(ctx(), owner, "owner-photo")
	if err != nil {
		t.Fatalf("GetPhotoData(owner): %v", err)
	}
	if string(got.Data) != "secret" {
		t.Fatalf("GetPhotoData(owner).Data = %q, want %q", got.Data, "secret")
	}

	// A different user requesting the same photo ID gets ErrNotFound, not the data.
	if _, err := s.GetPhotoData(ctx(), attacker, "owner-photo"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetPhotoData(attacker, owner's photo) = %v, want types.ErrNotFound", err)
	}
}

// TestGetPhotosData_CrossUserAccessDenied is the batch-fetch counterpart:
// a photo ID owned by a different user must be silently excluded from the
// result map (same as an unknown ID), not returned.
func TestGetPhotosData_CrossUserAccessDenied(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	owner := "u-owner-batch"
	attacker := "u-attacker-batch"
	mustUser(t, s, types.User{ID: owner})
	mustUser(t, s, types.User{ID: attacker})

	if err := s.UploadPhoto(ctx(), types.ProgressPhoto{
		ID: "owner-photo-2", UserID: owner, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("secret2"),
	}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}

	got, err := s.GetPhotosData(ctx(), attacker, []string{"owner-photo-2"})
	if err != nil {
		t.Fatalf("GetPhotosData(attacker): %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetPhotosData(attacker) = %v, want empty (owner's photo must not leak)", got)
	}

	// Confirm the owner can still fetch it — proves the empty result above is
	// due to user scoping, not the row being missing.
	got, err = s.GetPhotosData(ctx(), owner, []string{"owner-photo-2"})
	if err != nil {
		t.Fatalf("GetPhotosData(owner): %v", err)
	}
	if string(got["owner-photo-2"]) != "secret2" {
		t.Fatalf("GetPhotosData(owner)[owner-photo-2] = %q, want %q", got["owner-photo-2"], "secret2")
	}
}

func TestListPhotoMetadata_ExcludesBlobAndReturnsCorrectResults(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	uid := "u-photos-meta"
	mustUser(t, s, types.User{ID: uid})

	photos := []types.ProgressPhoto{
		{ID: "m1", UserID: uid, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("blob-1")},
		{ID: "m2", UserID: uid, Date: "2026-07-02", View: "side", MimeType: "image/jpeg", Data: []byte("blob-2")},
	}
	for _, p := range photos {
		if err := s.UploadPhoto(ctx(), p); err != nil {
			t.Fatalf("UploadPhoto(%s): %v", p.ID, err)
		}
	}

	got, err := s.ListPhotoMetadata(ctx(), uid)
	if err != nil {
		t.Fatalf("ListPhotoMetadata: %v", err)
	}
	if len(got) != len(photos) {
		t.Fatalf("ListPhotoMetadata returned %d rows, want %d", len(got), len(photos))
	}

	byID := make(map[string]types.ProgressPhoto, len(got))
	for _, p := range got {
		byID[p.ID] = p
	}
	for _, want := range photos {
		p, ok := byID[want.ID]
		if !ok {
			t.Fatalf("ListPhotoMetadata missing photo %s", want.ID)
		}
		if p.View != want.View || p.MimeType != want.MimeType || p.Date != want.Date {
			t.Fatalf("ListPhotoMetadata[%s] = %+v, want fields matching %+v", want.ID, p, want)
		}
		if len(p.Data) != 0 {
			t.Fatalf("ListPhotoMetadata[%s] included BLOB data %q, want none", want.ID, p.Data)
		}
	}
}

// TestDeletePhoto covers the SQL-layer delete used by handleDeletePhoto:
// a successful delete removes the row (and it's gone from later reads), a
// missing photo ID returns ErrNotFound, and deletion is scoped to the
// requesting user the same way reads are.
func TestDeletePhoto(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	uid := "u-photo-delete"
	other := "u-photo-delete-other"
	mustUser(t, s, types.User{ID: uid})
	mustUser(t, s, types.User{ID: other})

	if err := s.UploadPhoto(ctx(), types.ProgressPhoto{
		ID: "del-1", UserID: uid, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("bytes"),
	}); err != nil {
		t.Fatalf("UploadPhoto: %v", err)
	}

	// Wrong user for an existing photo ID: not found, and the row survives.
	if err := s.DeletePhoto(ctx(), other, "del-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("DeletePhoto(other user, del-1) = %v, want types.ErrNotFound", err)
	}
	if _, err := s.GetPhotoData(ctx(), uid, "del-1"); err != nil {
		t.Fatalf("photo deleted by wrong user's request: GetPhotoData: %v", err)
	}

	// Unknown photo ID: not found.
	if err := s.DeletePhoto(ctx(), uid, "no-such-photo"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("DeletePhoto(unknown id) = %v, want types.ErrNotFound", err)
	}

	// Owner deleting their own photo: succeeds, and the row is actually gone.
	if err := s.DeletePhoto(ctx(), uid, "del-1"); err != nil {
		t.Fatalf("DeletePhoto(owner): %v", err)
	}
	if _, err := s.GetPhotoData(ctx(), uid, "del-1"); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("GetPhotoData after delete = %v, want types.ErrNotFound", err)
	}
}

func TestCountPhotos(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()

	uid := "u-photo-count"
	mustUser(t, s, types.User{ID: uid})

	n, err := s.CountPhotos(ctx(), uid)
	if err != nil {
		t.Fatalf("CountPhotos (empty): %v", err)
	}
	if n != 0 {
		t.Fatalf("CountPhotos (empty) = %d, want 0", n)
	}

	for i := range 3 {
		if err := s.UploadPhoto(ctx(), types.ProgressPhoto{
			ID: fmt.Sprintf("count-%d", i), UserID: uid, Date: "2026-07-01", View: "front", MimeType: "image/png", Data: []byte("x"),
		}); err != nil {
			t.Fatalf("UploadPhoto(%d): %v", i, err)
		}
	}

	n, err = s.CountPhotos(ctx(), uid)
	if err != nil {
		t.Fatalf("CountPhotos: %v", err)
	}
	if n != 3 {
		t.Fatalf("CountPhotos = %d, want 3", n)
	}
}
