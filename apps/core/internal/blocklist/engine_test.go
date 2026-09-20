// SPDX-License-Identifier: GPL-3.0-or-later
package blocklist

import (
	"errors"
	"testing"

	"github.com/hydradns/hydradns/apps/core/internal/storage/repositories"
)

// fakeRepo embeds the (nil) interface and overrides only the methods under
// test here, so it satisfies repositories.BlocklistRepository without
// having to stub every method.
type fakeRepo struct {
	repositories.BlocklistRepository
	getAllEnabledCalled bool
	getAllCalled        bool
	signature           repositories.BlocklistSignature
	signatureErr        error
}

func (f *fakeRepo) GetAllEnabled() ([]string, error) {
	f.getAllEnabledCalled = true
	return []string{"enabled-only.example"}, nil
}

func (f *fakeRepo) GetAll() ([]string, error) {
	f.getAllCalled = true
	return []string{"enabled-only.example", "disabled.example"}, nil
}

func (f *fakeRepo) Signature() (repositories.BlocklistSignature, error) {
	return f.signature, f.signatureErr
}

// Engine.List must read from GetAllEnabled, not the unfiltered GetAll, or
// a disabled source's entries would still block. If this regresses back
// to GetAll, this test catches it.
func TestEngine_List_UsesGetAllEnabled(t *testing.T) {
	repo := &fakeRepo{}
	e := NewEngine(repo)

	domains, err := e.List()
	if err != nil {
		t.Fatal(err)
	}
	if !repo.getAllEnabledCalled {
		t.Error("List() did not call GetAllEnabled")
	}
	if repo.getAllCalled {
		t.Error("List() called the unfiltered GetAll — disabled sources would leak into the DNS hot path")
	}
	if len(domains) != 1 || domains[0] != "enabled-only.example" {
		t.Errorf("List() = %v, want [enabled-only.example]", domains)
	}
}

func TestEngine_Signature_DelegatesToRepo(t *testing.T) {
	want := repositories.BlocklistSignature{SourceCount: 3, EnabledSourceCount: 2}
	repo := &fakeRepo{signature: want}
	e := NewEngine(repo)

	got, err := e.Signature()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Signature() = %+v, want %+v", got, want)
	}
}

func TestEngine_Signature_PropagatesError(t *testing.T) {
	repo := &fakeRepo{signatureErr: errors.New("db down")}
	e := NewEngine(repo)

	if _, err := e.Signature(); err == nil {
		t.Error("expected Signature() to propagate the repo error")
	}
}
