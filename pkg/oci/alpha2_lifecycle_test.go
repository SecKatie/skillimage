package oci_test

import (
	"context"
	"testing"

	"github.com/redhat-et/skillimage/pkg/lifecycle"
	"github.com/redhat-et/skillimage/pkg/oci"
)

func TestAlpha2PromoteAndDemoteAllocateVersionsAndMoveLatest(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := client.Build(ctx, dir, oci.BuildOptions{}); err != nil {
		t.Fatal(err)
	}
	alphaRef := "localhost/pdf-processing:1.2.3-alpha.1"
	if err := client.PromoteStageLocal(ctx, alphaRef, lifecycle.Beta); err != nil {
		t.Fatalf("PromoteStageLocal: %v", err)
	}
	betaRef := "localhost/pdf-processing:1.2.3-beta.1"
	for _, ref := range []string{betaRef, "localhost/pdf-processing:latest"} {
		result, err := client.Inspect(ctx, ref)
		if err != nil {
			t.Fatalf("Inspect(%s): %v", ref, err)
		}
		if result.Version != "1.2.3-beta.1" || result.Status != "beta" {
			t.Fatalf("Inspect(%s) = %#v", ref, result)
		}
	}
	if err := client.DemoteStageLocal(ctx, betaRef, lifecycle.Alpha); err != nil {
		t.Fatalf("DemoteStageLocal: %v", err)
	}
	result, err := client.Inspect(ctx, "localhost/pdf-processing:1.2.3-alpha.2")
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "1.2.3-alpha.2" || result.Status != "alpha" {
		t.Fatalf("unexpected demoted result: %#v", result)
	}
}

func TestAlpha2LifecycleDirectionIsStrict(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := client.Build(ctx, dir, oci.BuildOptions{}); err != nil {
		t.Fatal(err)
	}
	ref := "localhost/pdf-processing:1.2.3-alpha.1"
	if err := client.PromoteStageLocal(ctx, ref, lifecycle.Alpha); err == nil {
		t.Fatal("promote must reject same or lower stage")
	}
	if err := client.DemoteStageLocal(ctx, ref, lifecycle.Beta); err == nil {
		t.Fatal("demote must reject same or higher stage")
	}
}

func TestAlpha2PromotionDoesNotInventLatestForExactBuild(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ref := "localhost/pdf-processing:canary"
	if _, err := client.Build(ctx, dir, oci.BuildOptions{Tag: ref}); err != nil {
		t.Fatal(err)
	}
	if err := client.PromoteStageLocal(ctx, ref, lifecycle.Beta); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Inspect(ctx, "localhost/pdf-processing:latest"); err == nil {
		t.Fatal("promotion should not create latest when latest did not point to source")
	}
	if result, err := client.Inspect(ctx, "localhost/pdf-processing:1.2.3-beta.1"); err != nil || result.Status != "beta" {
		t.Fatalf("expected beta promotion, got %#v, %v", result, err)
	}
}

func TestAlpha2PromotionToFinalUsesBaseVersion(t *testing.T) {
	dir := writeAlpha2Skill(t, t.TempDir(), "pdf-processing")
	client, err := oci.NewClient(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := client.Build(ctx, dir, oci.BuildOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := client.PromoteStageLocal(ctx, "localhost/pdf-processing:latest", lifecycle.Final); err != nil {
		t.Fatal(err)
	}
	result, err := client.Inspect(ctx, "localhost/pdf-processing:1.2.3")
	if err != nil || result.Version != "1.2.3" || result.Status != "final" {
		t.Fatalf("unexpected final result: %#v, %v", result, err)
	}
	if _, err := client.Build(ctx, dir, oci.BuildOptions{}); err == nil {
		t.Fatal("expected a new default build after final to require a base version bump")
	}
}
