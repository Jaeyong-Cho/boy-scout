package abstractness

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCheck_FlagsZoneOfPain tests that a concrete stable package is flagged as Pain.
// Given: legacydb (6 exported structs, 0 interfaces → A=0.0; Ca=8, Ce=0 → I=0.0)
// Expected: Zone="Pain", Distance=1.0
func TestCheck_FlagsZoneOfPain(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/legacy\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create legacydb package: 6 exported structs, no interfaces
	legacydbDir := filepath.Join(tempDir, "legacydb")
	os.Mkdir(legacydbDir, 0755)

	legacydbCode := `package legacydb

type UserRow struct{ ID int }
type OrderRow struct{ ID int }
type ProductRow struct{ ID int }
type InvoiceRow struct{ ID int }
type LogEntry struct{ ID int }
type ConfigEntry struct{ ID int }

func Query() {}
`

	if err := os.WriteFile(filepath.Join(legacydbDir, "legacydb.go"), []byte(legacydbCode), 0644); err != nil {
		t.Fatalf("failed to write legacydb.go: %v", err)
	}

	// Create 8 caller packages importing legacydb
	for i := 1; i <= 8; i++ {
		callerDir := filepath.Join(tempDir, fmt.Sprintf("caller%d", i))
		os.Mkdir(callerDir, 0755)
		callerCode := fmt.Sprintf(`package caller%d

import "example.com/legacy/legacydb"

func Call%d() {
	legacydb.Query()
}
`, i, i)
		if err := os.WriteFile(filepath.Join(callerDir, fmt.Sprintf("caller%d.go", i)), []byte(callerCode), 0644); err != nil {
			t.Fatalf("failed to write caller%d: %v", i, err)
		}
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the legacydb violation
	var diagnosis *PackageDiagnosis
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/legacy/legacydb" {
			diagnosis = &report.Violations[i]
			break
		}
	}

	if diagnosis == nil {
		t.Fatalf("expected legacydb in violations, got: %v", report.Violations)
	}

	if diagnosis.Zone != "Pain" {
		t.Fatalf("expected Zone=Pain, got %s", diagnosis.Zone)
	}

	// A=0.0, I=0.0, signedD = 0.0 + 0.0 - 1.0 = -1.0, Distance = 1.0
	if fmt.Sprintf("%.1f", diagnosis.Abstractness) != "0.0" {
		t.Fatalf("expected Abstractness≈0.0, got %.1f", diagnosis.Abstractness)
	}

	if fmt.Sprintf("%.1f", diagnosis.Distance) != "1.0" {
		t.Fatalf("expected Distance≈1.0, got %.1f", diagnosis.Distance)
	}
}

// TestCheck_FlagsZoneOfUselessness tests that an abstract unstable package is flagged as Uselessness.
// Given: plugini (3 exported interfaces, 0 structs → A=1.0; Ca=0, Ce=3 → I=1.0)
// Expected: Zone="Uselessness", Distance=1.0
func TestCheck_FlagsZoneOfUselessness(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/plugin\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create plugini package: 3 exported interfaces, no structs
	pluginiDir := filepath.Join(tempDir, "plugini")
	os.Mkdir(pluginiDir, 0755)

	pluginiCode := `package plugini

type Handler interface {
	Handle(msg string) error
}

type Middleware interface {
	Next(fn func() error) error
}

type Logger interface {
	Log(msg string)
}
`

	if err := os.WriteFile(filepath.Join(pluginiDir, "plugini.go"), []byte(pluginiCode), 0644); err != nil {
		t.Fatalf("failed to write plugini.go: %v", err)
	}

	// Create 3 packages for plugini to import
	for i := 1; i <= 3; i++ {
		depDir := filepath.Join(tempDir, fmt.Sprintf("dep%d", i))
		os.Mkdir(depDir, 0755)
		depCode := fmt.Sprintf(`package dep%d

func Dep%d() {}
`, i, i)
		if err := os.WriteFile(filepath.Join(depDir, fmt.Sprintf("dep%d.go", i)), []byte(depCode), 0644); err != nil {
			t.Fatalf("failed to write dep%d: %v", i, err)
		}
	}

	// plugini imports 3 packages
	importsCode := `package plugini

import (
	"example.com/plugin/dep1"
	"example.com/plugin/dep2"
	"example.com/plugin/dep3"
)

func init() {
	_ = dep1.Dep1
	_ = dep2.Dep2
	_ = dep3.Dep3
}
`
	if err := os.WriteFile(filepath.Join(pluginiDir, "plugini_imports.go"), []byte(importsCode), 0644); err != nil {
		t.Fatalf("failed to write plugini_imports.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the plugini violation
	var diagnosis *PackageDiagnosis
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/plugin/plugini" {
			diagnosis = &report.Violations[i]
			break
		}
	}

	if diagnosis == nil {
		t.Fatalf("expected plugini in violations, got: %v", report.Violations)
	}

	if diagnosis.Zone != "Uselessness" {
		t.Fatalf("expected Zone=Uselessness, got %s", diagnosis.Zone)
	}

	// A=1.0, I=1.0, signedD = 1.0 + 1.0 - 1.0 = 1.0, Distance = 1.0
	if fmt.Sprintf("%.1f", diagnosis.Abstractness) != "1.0" {
		t.Fatalf("expected Abstractness≈1.0, got %.1f", diagnosis.Abstractness)
	}

	if fmt.Sprintf("%.1f", diagnosis.Distance) != "1.0" {
		t.Fatalf("expected Distance≈1.0, got %.1f", diagnosis.Distance)
	}
}

// TestCheck_NoViolationWhenAbstractAndStable tests that abstract stable packages are not flagged.
// Given: stableiface (all exported types are interfaces → A=1.0; Ca=5, Ce=0 → I=0.0, signedD=0)
// Expected: not in violations
func TestCheck_NoViolationWhenAbstractAndStable(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/stable\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create stableiface: all exported types are interfaces
	stableDir := filepath.Join(tempDir, "stableiface")
	os.Mkdir(stableDir, 0755)

	stableCode := `package stableiface

type Service interface {
	Do() error
}

type Provider interface {
	Provide() Service
}
`

	if err := os.WriteFile(filepath.Join(stableDir, "stableiface.go"), []byte(stableCode), 0644); err != nil {
		t.Fatalf("failed to write stableiface.go: %v", err)
	}

	// Create 5 packages importing stableiface (Ca=5, Ce=0, I=0.0)
	for i := 1; i <= 5; i++ {
		clientDir := filepath.Join(tempDir, fmt.Sprintf("client%d", i))
		os.Mkdir(clientDir, 0755)
		clientCode := fmt.Sprintf(`package client%d

import "example.com/stable/stableiface"

func Call%d(svc stableiface.Service) error {
	return svc.Do()
}
`, i, i)
		if err := os.WriteFile(filepath.Join(clientDir, fmt.Sprintf("client%d.go", i)), []byte(clientCode), 0644); err != nil {
			t.Fatalf("failed to write client%d: %v", i, err)
		}
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that stableiface is NOT in violations
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/stable/stableiface" {
			t.Fatalf("stableiface should not be flagged (ideal: abstract and stable)")
		}
	}
}

// TestCheck_NoViolationWhenConcreteAndUnstable tests that concrete unstable packages are not flagged.
// Given: leafimpl (0 interfaces, some structs → A=0.0; Ca=0, Ce=4 → I=1.0, signedD=0)
// Expected: not in violations
func TestCheck_NoViolationWhenConcreteAndUnstable(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/leaf\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create leafimpl: concrete (no interfaces), unstable (imports 4, nothing imports it)
	leafDir := filepath.Join(tempDir, "leafimpl")
	os.Mkdir(leafDir, 0755)

	leafCode := `package leafimpl

type Config struct {
	Key   string
	Value string
}

type State struct {
	Data map[string]interface{}
}
`

	if err := os.WriteFile(filepath.Join(leafDir, "leafimpl.go"), []byte(leafCode), 0644); err != nil {
		t.Fatalf("failed to write leafimpl.go: %v", err)
	}

	// Create 4 packages for leafimpl to import
	for i := 1; i <= 4; i++ {
		depDir := filepath.Join(tempDir, fmt.Sprintf("dep%d", i))
		os.Mkdir(depDir, 0755)
		depCode := fmt.Sprintf(`package dep%d

func Dep%d() {}
`, i, i)
		if err := os.WriteFile(filepath.Join(depDir, fmt.Sprintf("dep%d.go", i)), []byte(depCode), 0644); err != nil {
			t.Fatalf("failed to write dep%d: %v", i, err)
		}
	}

	// leafimpl imports 4 packages
	importsCode := `package leafimpl

import (
	"example.com/leaf/dep1"
	"example.com/leaf/dep2"
	"example.com/leaf/dep3"
	"example.com/leaf/dep4"
)

func init() {
	_ = dep1.Dep1
	_ = dep2.Dep2
	_ = dep3.Dep3
	_ = dep4.Dep4
}
`
	if err := os.WriteFile(filepath.Join(leafDir, "leafimpl_imports.go"), []byte(importsCode), 0644); err != nil {
		t.Fatalf("failed to write leafimpl_imports.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that leafimpl is NOT in violations
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/leaf/leafimpl" {
			t.Fatalf("leafimpl should not be flagged (ideal: concrete and unstable)")
		}
	}
}

// TestCheck_ExactlyAtMinDistanceIsCompliant tests the boundary where signedD is exactly at -minDistance.
// Given: minDistance=0.5 and a package with signedD=-0.5 (exactly at boundary)
// Expected: not flagged (strict < comparison)
func TestCheck_ExactlyAtMinDistanceIsCompliant(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/boundary\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create a package with A=0.0, I=0.5 => signedD = 0.0 + 0.5 - 1.0 = -0.5
	// To get I=0.5: Ca=1, Ce=1
	boundaryDir := filepath.Join(tempDir, "boundary")
	os.Mkdir(boundaryDir, 0755)

	boundaryCode := `package boundary

type Thing struct{}

func Do() {}
`

	if err := os.WriteFile(filepath.Join(boundaryDir, "boundary.go"), []byte(boundaryCode), 0644); err != nil {
		t.Fatalf("failed to write boundary.go: %v", err)
	}

	// Create one package for boundary to import
	depDir := filepath.Join(tempDir, "dep")
	os.Mkdir(depDir, 0755)
	if err := os.WriteFile(filepath.Join(depDir, "dep.go"), []byte("package dep\nfunc Dep() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write dep.go: %v", err)
	}

	// Make boundary import dep (Ce=1) and be imported by one caller (Ca=1)
	importsCode := `package boundary

import "example.com/boundary/dep"

func init() {
	_ = dep.Dep
}
`
	if err := os.WriteFile(filepath.Join(boundaryDir, "boundary_imports.go"), []byte(importsCode), 0644); err != nil {
		t.Fatalf("failed to write boundary_imports.go: %v", err)
	}

	// Create a caller importing boundary
	callerDir := filepath.Join(tempDir, "caller")
	os.Mkdir(callerDir, 0755)
	callerCode := `package caller

import "example.com/boundary/boundary"

func Call() {
	boundary.Do()
}
`
	if err := os.WriteFile(filepath.Join(callerDir, "caller.go"), []byte(callerCode), 0644); err != nil {
		t.Fatalf("failed to write caller.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that boundary is NOT flagged (exactly at -minDistance boundary)
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/boundary/boundary" {
			t.Fatalf("package at exactly -minDistance should not be flagged (strict <)")
		}
	}
}

// TestCheck_ZeroExportedTypesUsesAZero tests packages with zero exported types.
// Given: a package with no exported interfaces/structs
// Expected: A=0 (no divide-by-zero error)
func TestCheck_ZeroExportedTypesUsesAZero(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/notypes\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create a package with no exported types (pure functions)
	noTypesDir := filepath.Join(tempDir, "notypes")
	os.Mkdir(noTypesDir, 0755)

	noTypesCode := `package notypes

func helper1() {}
func helper2() {}
func helper3() {}
`

	if err := os.WriteFile(filepath.Join(noTypesDir, "notypes.go"), []byte(noTypesCode), 0644); err != nil {
		t.Fatalf("failed to write notypes.go: %v", err)
	}

	// Create a caller importing it (to make it part of a graph)
	callerDir := filepath.Join(tempDir, "caller")
	os.Mkdir(callerDir, 0755)
	callerCode := `package caller

import "example.com/notypes/notypes"

func init() {
	_ = notypes.helper1
}
`
	if err := os.WriteFile(filepath.Join(callerDir, "caller.go"), []byte(callerCode), 0644); err != nil {
		t.Fatalf("failed to write caller.go: %v", err)
	}

	// notypes also imports something to create an edge
	depDir := filepath.Join(tempDir, "dep")
	os.Mkdir(depDir, 0755)
	if err := os.WriteFile(filepath.Join(depDir, "dep.go"), []byte("package dep\nfunc Dep() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write dep.go: %v", err)
	}

	importsCode := `package notypes

import "example.com/notypes/dep"

func init() {
	_ = dep.Dep
}
`
	if err := os.WriteFile(filepath.Join(noTypesDir, "notypes_imports.go"), []byte(importsCode), 0644); err != nil {
		t.Fatalf("failed to write notypes_imports.go: %v", err)
	}

	// Should not panic on divide-by-zero
	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the notypes package
	var diagnosis *PackageDiagnosis
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/notypes/notypes" {
			diagnosis = &report.Violations[i]
			break
		}
	}

	// If it's flagged, check that Abstractness=0
	if diagnosis != nil {
		if diagnosis.Abstractness != 0.0 {
			t.Fatalf("expected Abstractness=0.0 for zero exported types, got %f", diagnosis.Abstractness)
		}
	}
}

// TestCheck_IsolatedPackageSkipped tests that packages with no internal edges are skipped.
// Given: a package with Ca=Ce=0 (isolated)
// Expected: does not appear in violations or TotalPackages count
func TestCheck_IsolatedPackageSkipped(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/isolated\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create one package with no imports and no importers (isolated)
	isolatedDir := filepath.Join(tempDir, "isolated")
	os.Mkdir(isolatedDir, 0755)

	isolatedCode := `package isolated

type IsolatedType struct{}

func Do() {}
`

	if err := os.WriteFile(filepath.Join(isolatedDir, "isolated.go"), []byte(isolatedCode), 0644); err != nil {
		t.Fatalf("failed to write isolated.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Isolated package should not appear in violations
	if len(report.Violations) > 0 {
		t.Fatalf("isolated package should not be flagged, got: %v", report.Violations)
	}

	// TotalPackages should be 0 (no packages with edges)
	if report.TotalPackages != 0 {
		t.Fatalf("expected TotalPackages=0 for isolated package, got %d", report.TotalPackages)
	}
}

// TestCheck_SkipsUnparsableFile tests that unparsable files are skipped gracefully.
// Given: a package with one unparsable file
// Expected: skipped record added, rest of module still checked
func TestCheck_SkipsUnparsableFile(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/parseable\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create a package with an unparsable file
	pkgDir := filepath.Join(tempDir, "pkg")
	os.Mkdir(pkgDir, 0755)

	validCode := `package pkg

type Thing struct{}

func Do() {}
`

	if err := os.WriteFile(filepath.Join(pkgDir, "valid.go"), []byte(validCode), 0644); err != nil {
		t.Fatalf("failed to write valid.go: %v", err)
	}

	// Write an unparsable file
	brokenCode := `package pkg

type Broken struct {
	// missing closing brace
`

	if err := os.WriteFile(filepath.Join(pkgDir, "broken.go"), []byte(brokenCode), 0644); err != nil {
		t.Fatalf("failed to write broken.go: %v", err)
	}

	// Create a caller to put pkg in the graph
	callerDir := filepath.Join(tempDir, "caller")
	os.Mkdir(callerDir, 0755)
	callerCode := `package caller

import "example.com/parseable/pkg"

func Call() {
	pkg.Do()
}
`
	if err := os.WriteFile(filepath.Join(callerDir, "caller.go"), []byte(callerCode), 0644); err != nil {
		t.Fatalf("failed to write caller.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have skipped the broken file
	if len(report.Skipped) == 0 {
		t.Fatal("expected at least one skipped file")
	}

	found := false
	for _, skipped := range report.Skipped {
		if filepath.Base(skipped.File) == "broken.go" {
			found = true
			if skipped.Error == "" {
				t.Fatal("expected skipped file to have an error message")
			}
			break
		}
	}

	if !found {
		t.Fatal("expected broken.go in skipped files")
	}
}

// TestCheck_ShallowPainStillFlagged tests that a concrete stable package with SurfaceRatio=1.0 is still flagged.
// Given: legacydb (6 exported structs, 0 unexported → SurfaceRatio=1.0)
// When: abstractness.Check runs with default minSurfaceRatio=0.5
// Then: it is flagged as Zone of Pain (shallow — nothing hidden)
func TestCheck_ShallowPainStillFlagged(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/legacy\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create legacydb package: 6 exported structs, no unexported helpers
	legacydbDir := filepath.Join(tempDir, "legacydb")
	os.Mkdir(legacydbDir, 0755)

	legacydbCode := `package legacydb

type UserRow struct{ ID int }
type OrderRow struct{ ID int }
type ProductRow struct{ ID int }
type InvoiceRow struct{ ID int }
type LogEntry struct{ ID int }
type ConfigEntry struct{ ID int }

func Query() {}
`

	if err := os.WriteFile(filepath.Join(legacydbDir, "legacydb.go"), []byte(legacydbCode), 0644); err != nil {
		t.Fatalf("failed to write legacydb.go: %v", err)
	}

	// Create 8 caller packages importing legacydb
	for i := 1; i <= 8; i++ {
		callerDir := filepath.Join(tempDir, fmt.Sprintf("caller%d", i))
		os.Mkdir(callerDir, 0755)
		callerCode := fmt.Sprintf(`package caller%d

import "example.com/legacy/legacydb"

func Call%d() {
	legacydb.Query()
}
`, i, i)
		if err := os.WriteFile(filepath.Join(callerDir, fmt.Sprintf("caller%d.go", i)), []byte(callerCode), 0644); err != nil {
			t.Fatalf("failed to write caller%d: %v", i, err)
		}
	}

	report, err := CheckWithSurfaceRatio([]string{tempDir}, 0.5, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the legacydb violation
	var diagnosis *PackageDiagnosis
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/legacy/legacydb" {
			diagnosis = &report.Violations[i]
			break
		}
	}

	if diagnosis == nil {
		t.Fatalf("expected legacydb in violations, got: %v", report.Violations)
	}

	if diagnosis.Zone != "Pain" {
		t.Fatalf("expected Zone=Pain, got %s", diagnosis.Zone)
	}

	if fmt.Sprintf("%.1f", diagnosis.SurfaceRatio) != "1.0" {
		t.Fatalf("expected SurfaceRatio≈1.0 for legacydb, got %.1f", diagnosis.SurfaceRatio)
	}
}

// TestCheck_DeepPainCandidateNotFlagged tests that a deep module (small interface, lots of hidden impl) is NOT flagged.
// Given: deepcache (3 exported structs, ~40 unexported helpers → SurfaceRatio≈0.07; Ca=8, Ce=0 → I=0.0, Pain candidate)
// When: abstractness.Check runs with default minSurfaceRatio=0.5
// Then: it is NOT flagged (deep module — concrete+stable is fine when little is exposed)
func TestCheck_DeepPainCandidateNotFlagged(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/cache\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create deepcache package: 3 exported structs, lots of unexported helpers
	deepcacheDir := filepath.Join(tempDir, "deepcache")
	os.Mkdir(deepcacheDir, 0755)

	// Create a deep module: small public interface, lots of hidden implementation
	deepcacheCode := `package deepcache

// Public interface
type Cache struct {
	data map[string]interface{}
	lock interface{}
}

type Config struct {
	MaxSize int
}

type Stats struct {
	Hits   int
	Misses int
}

// ~40 unexported helper functions and types
func (c *Cache) get(key string) interface{} { return nil }
func (c *Cache) set(key string, val interface{}) {}
func (c *Cache) evict(key string) {}
func (c *Cache) findLRU() string { return "" }
func (c *Cache) updateLRU(key string) {}
func (c *Cache) marshal(v interface{}) []byte { return nil }
func (c *Cache) unmarshal(data []byte) interface{} { return nil }
func (c *Cache) compress(data []byte) []byte { return nil }
func (c *Cache) decompress(data []byte) []byte { return nil }

type lruNode struct{ key string }
type hashBucket struct{ items []lruNode }
type serializer interface{}
type compressor interface{}

func newLRUNode(key string) *lruNode { return nil }
func newHashBucket() *hashBucket { return nil }
func newSerializer() serializer { return nil }
func newCompressor() compressor { return nil }
func hashKey(key string) uint64 { return 0 }
func validateKey(key string) error { return nil }
func validateSize(sz int) error { return nil }
func computeHash(data []byte) uint64 { return 0 }
func encodeStats(s *Stats) []byte { return nil }
func decodeStats(data []byte) *Stats { return nil }
func diffStats(s1, s2 *Stats) *Stats { return nil }
func mergeStats(s1, s2 *Stats) *Stats { return nil }
`

	if err := os.WriteFile(filepath.Join(deepcacheDir, "deepcache.go"), []byte(deepcacheCode), 0644); err != nil {
		t.Fatalf("failed to write deepcache.go: %v", err)
	}

	// Create 8 caller packages importing deepcache (Ca=8, Ce=0 → I=0.0)
	for i := 1; i <= 8; i++ {
		callerDir := filepath.Join(tempDir, fmt.Sprintf("caller%d", i))
		os.Mkdir(callerDir, 0755)
		callerCode := fmt.Sprintf(`package caller%d

import "example.com/cache/deepcache"

func Call%d(cfg deepcache.Config) {
	_ = cfg
}
`, i, i)
		if err := os.WriteFile(filepath.Join(callerDir, fmt.Sprintf("caller%d.go", i)), []byte(callerCode), 0644); err != nil {
			t.Fatalf("failed to write caller%d: %v", i, err)
		}
	}

	report, err := CheckWithSurfaceRatio([]string{tempDir}, 0.5, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// deepcache should NOT be in violations (it's a deep module)
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/cache/deepcache" {
			t.Fatalf("deepcache should not be flagged (deep module), got SurfaceRatio=%f", report.Violations[i].SurfaceRatio)
		}
	}
}

// TestCheck_IgnoreDeepModuleGateRestoresOldBehavior tests that --ignore-deep-module-gate restores Story 2 behavior.
// Given: deepcache fixture with SurfaceRatio≈0.07
// When: abstractness.Check runs with IgnoreDeepModuleGate=true
// Then: it IS flagged as Zone of Pain (old Story 2 behavior, for comparison)
func TestCheck_IgnoreDeepModuleGateRestoresOldBehavior(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/cache\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create deepcache package (same as above)
	deepcacheDir := filepath.Join(tempDir, "deepcache")
	os.Mkdir(deepcacheDir, 0755)

	deepcacheCode := `package deepcache

type Cache struct {
	data map[string]interface{}
	lock interface{}
}

type Config struct {
	MaxSize int
}

type Stats struct {
	Hits   int
	Misses int
}

func (c *Cache) get(key string) interface{} { return nil }
func (c *Cache) set(key string, val interface{}) {}
func (c *Cache) evict(key string) {}
func (c *Cache) findLRU() string { return "" }
func (c *Cache) updateLRU(key string) {}
func (c *Cache) marshal(v interface{}) []byte { return nil }
func (c *Cache) unmarshal(data []byte) interface{} { return nil }
func (c *Cache) compress(data []byte) []byte { return nil }
func (c *Cache) decompress(data []byte) []byte { return nil }

type lruNode struct{ key string }
type hashBucket struct{ items []lruNode }
type serializer interface{}
type compressor interface{}

func newLRUNode(key string) *lruNode { return nil }
func newHashBucket() *hashBucket { return nil }
func newSerializer() serializer { return nil }
func newCompressor() compressor { return nil }
func hashKey(key string) uint64 { return 0 }
func validateKey(key string) error { return nil }
func validateSize(sz int) error { return nil }
func computeHash(data []byte) uint64 { return 0 }
func encodeStats(s *Stats) []byte { return nil }
func decodeStats(data []byte) *Stats { return nil }
func diffStats(s1, s2 *Stats) *Stats { return nil }
func mergeStats(s1, s2 *Stats) *Stats { return nil }
`

	if err := os.WriteFile(filepath.Join(deepcacheDir, "deepcache.go"), []byte(deepcacheCode), 0644); err != nil {
		t.Fatalf("failed to write deepcache.go: %v", err)
	}

	// Create 8 caller packages
	for i := 1; i <= 8; i++ {
		callerDir := filepath.Join(tempDir, fmt.Sprintf("caller%d", i))
		os.Mkdir(callerDir, 0755)
		callerCode := fmt.Sprintf(`package caller%d

import "example.com/cache/deepcache"

func Call%d(cfg deepcache.Config) {
	_ = cfg
}
`, i, i)
		if err := os.WriteFile(filepath.Join(callerDir, fmt.Sprintf("caller%d.go", i)), []byte(callerCode), 0644); err != nil {
			t.Fatalf("failed to write caller%d: %v", i, err)
		}
	}

	// Run with IgnoreDeepModuleGate=true
	report, err := CheckWithSurfaceRatio([]string{tempDir}, 0.5, 0.5, Options{IgnoreDeepModuleGate: true})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// deepcache SHOULD be in violations now (old behavior)
	var diagnosis *PackageDiagnosis
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/cache/deepcache" {
			diagnosis = &report.Violations[i]
			break
		}
	}

	if diagnosis == nil {
		t.Fatalf("expected deepcache in violations with IgnoreDeepModuleGate=true, got: %v", report.Violations)
	}

	if diagnosis.Zone != "Pain" {
		t.Fatalf("expected Zone=Pain, got %s", diagnosis.Zone)
	}
}

// TestCheck_ExactlyAtMinSurfaceRatioIsFlagged tests the boundary where SurfaceRatio=minSurfaceRatio.
// Given: a package with SurfaceRatio exactly 0.5 at default minSurfaceRatio=0.5, and also a Pain candidate (signedD < -0.5)
// When: checked
// Then: it IS flagged (boundary: gate uses >=, exactly-at-limit still counts as shallow enough)
func TestCheck_ExactlyAtMinSurfaceRatioIsFlagged(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/boundary\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create a package with SurfaceRatio exactly 0.5 (3 exported, 3 unexported top-level declarations)
	// Also make it a Pain candidate: A=0 (concrete), I < 0.5 (stable/selective imports)
	boundaryDir := filepath.Join(tempDir, "boundary")
	os.Mkdir(boundaryDir, 0755)

	// All declarations in one file to have exact control over the count
	boundaryCode := `package boundary

import "example.com/boundary/dep"

// Exported
type Thing struct{}
type Stuff struct{}
func Do() { _ = dep.Dep }

// Unexported
type thing struct{}
type stuff struct{}
func helper() {}
`

	if err := os.WriteFile(filepath.Join(boundaryDir, "boundary.go"), []byte(boundaryCode), 0644); err != nil {
		t.Fatalf("failed to write boundary.go: %v", err)
	}

	// Create one package for boundary to import (Ce=1)
	depDir := filepath.Join(tempDir, "dep")
	os.Mkdir(depDir, 0755)
	if err := os.WriteFile(filepath.Join(depDir, "dep.go"), []byte("package dep\nfunc Dep() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write dep.go: %v", err)
	}

	// Create 5 callers importing boundary (Ca=5)
	// This gives I = Ce / (Ca + Ce) = 1 / 6 ≈ 0.167 < 0.5 → Pain
	// signedD = 0 + 0.167 - 1 = -0.833 < -0.5 ✓
	for i := 1; i <= 5; i++ {
		callerDir := filepath.Join(tempDir, fmt.Sprintf("caller%d", i))
		os.Mkdir(callerDir, 0755)
		callerCode := fmt.Sprintf(`package caller%d

import "example.com/boundary/boundary"

func Call%d() {
	boundary.Do()
}
`, i, i)
		if err := os.WriteFile(filepath.Join(callerDir, fmt.Sprintf("caller%d.go", i)), []byte(callerCode), 0644); err != nil {
			t.Fatalf("failed to write caller%d: %v", i, err)
		}
	}

	report, err := CheckWithSurfaceRatio([]string{tempDir}, 0.5, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// boundary should be flagged (SurfaceRatio=0.5 >= minSurfaceRatio=0.5 and it's a Pain candidate)
	var diagnosis *PackageDiagnosis
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/boundary/boundary" {
			diagnosis = &report.Violations[i]
			break
		}
	}

	if diagnosis == nil {
		t.Fatalf("expected boundary in violations (SurfaceRatio at boundary), got: %v", report.Violations)
	}

	if diagnosis.Zone != "Pain" {
		t.Fatalf("expected Zone=Pain, got %s", diagnosis.Zone)
	}

	if fmt.Sprintf("%.1f", diagnosis.SurfaceRatio) != "0.5" {
		t.Fatalf("expected SurfaceRatio≈0.5, got %.1f", diagnosis.SurfaceRatio)
	}
}

// TestCheck_UselessnessUnaffectedByGate tests that Uselessness is NOT gated by SurfaceRatio.
// Given: plugini fixture (Zone of Uselessness) with any SurfaceRatio
// When: checked
// Then: it is still flagged exactly as Story 2 did (gate does not apply to Uselessness)
func TestCheck_UselessnessUnaffectedByGate(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/plugin\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create plugini package: 3 exported interfaces (A=1.0)
	pluginiDir := filepath.Join(tempDir, "plugini")
	os.Mkdir(pluginiDir, 0755)

	pluginiCode := `package plugini

type Handler interface {
	Handle(msg string) error
}

type Middleware interface {
	Next(fn func() error) error
}

type Logger interface {
	Log(msg string)
}
`

	if err := os.WriteFile(filepath.Join(pluginiDir, "plugini.go"), []byte(pluginiCode), 0644); err != nil {
		t.Fatalf("failed to write plugini.go: %v", err)
	}

	// Create 3 packages for plugini to import (I=1.0)
	for i := 1; i <= 3; i++ {
		depDir := filepath.Join(tempDir, fmt.Sprintf("dep%d", i))
		os.Mkdir(depDir, 0755)
		depCode := fmt.Sprintf(`package dep%d

func Dep%d() {}
`, i, i)
		if err := os.WriteFile(filepath.Join(depDir, fmt.Sprintf("dep%d.go", i)), []byte(depCode), 0644); err != nil {
			t.Fatalf("failed to write dep%d: %v", i, err)
		}
	}

	// plugini imports 3 packages
	importsCode := `package plugini

import (
	"example.com/plugin/dep1"
	"example.com/plugin/dep2"
	"example.com/plugin/dep3"
)

func init() {
	_ = dep1.Dep1
	_ = dep2.Dep2
	_ = dep3.Dep3
}
`
	if err := os.WriteFile(filepath.Join(pluginiDir, "plugini_imports.go"), []byte(importsCode), 0644); err != nil {
		t.Fatalf("failed to write plugini_imports.go: %v", err)
	}

	report, err := CheckWithSurfaceRatio([]string{tempDir}, 0.5, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the plugini violation
	var diagnosis *PackageDiagnosis
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/plugin/plugini" {
			diagnosis = &report.Violations[i]
			break
		}
	}

	if diagnosis == nil {
		t.Fatalf("expected plugini in violations, got: %v", report.Violations)
	}

	if diagnosis.Zone != "Uselessness" {
		t.Fatalf("expected Zone=Uselessness, got %s", diagnosis.Zone)
	}

	// Verify SurfaceRatio=0 for Uselessness row (postcondition)
	if diagnosis.SurfaceRatio != 0 {
		t.Fatalf("expected SurfaceRatio=0 for Uselessness, got %f", diagnosis.SurfaceRatio)
	}
}

// TestCheck_TestFuncsNotCountedAsExported tests that TestXxx/BenchmarkXxx/ExampleXxx functions in _test.go files
// are not counted as exported symbols, even though they start with a capital letter.
// Given: deepcache (3 exported types: Cache, Config, Stats; ~21 unexported helpers; 25 TestBehaviorN functions in _test.go)
// When: abstractness.CheckWithSurfaceRatio runs with minSurfaceRatio=0.5
// Then: deepcache is NOT flagged (test funcs excluded, real SurfaceRatio stays under gate)
func TestCheck_TestFuncsNotCountedAsExported(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/cache\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create deepcache package: 3 exported types, ~21 unexported helpers
	deepcacheDir := filepath.Join(tempDir, "deepcache")
	os.Mkdir(deepcacheDir, 0755)

	// Public interface with 3 exported structs
	deepcacheCode := `package deepcache

type Cache struct {
	data map[string]interface{}
	lock interface{}
}

type Config struct {
	MaxSize int
}

type Stats struct {
	Hits   int
	Misses int
}

// ~21 unexported helper functions and types
func (c *Cache) get(key string) interface{} { return nil }
func (c *Cache) set(key string, val interface{}) {}
func (c *Cache) evict(key string) {}
func (c *Cache) findLRU() string { return "" }
func (c *Cache) updateLRU(key string) {}
func (c *Cache) marshal(v interface{}) []byte { return nil }
func (c *Cache) unmarshal(data []byte) interface{} { return nil }
func (c *Cache) compress(data []byte) []byte { return nil }
func (c *Cache) decompress(data []byte) []byte { return nil }

type lruNode struct{ key string }
type hashBucket struct{ items []lruNode }
type serializer interface{}
type compressor interface{}

func newLRUNode(key string) *lruNode { return nil }
func newHashBucket() *hashBucket { return nil }
func newSerializer() serializer { return nil }
func newCompressor() compressor { return nil }
func hashKey(key string) uint64 { return 0 }
func validateKey(key string) error { return nil }
func validateSize(sz int) error { return nil }
func computeHash(data []byte) uint64 { return 0 }
`

	if err := os.WriteFile(filepath.Join(deepcacheDir, "deepcache.go"), []byte(deepcacheCode), 0644); err != nil {
		t.Fatalf("failed to write deepcache.go: %v", err)
	}

	// Create deepcache_test.go with 25 TestBehaviorN functions
	// These should NOT be counted as exported symbols
	testCode := `package deepcache

import "testing"

func TestBehavior1(t *testing.T) {}
func TestBehavior2(t *testing.T) {}
func TestBehavior3(t *testing.T) {}
func TestBehavior4(t *testing.T) {}
func TestBehavior5(t *testing.T) {}
func TestBehavior6(t *testing.T) {}
func TestBehavior7(t *testing.T) {}
func TestBehavior8(t *testing.T) {}
func TestBehavior9(t *testing.T) {}
func TestBehavior10(t *testing.T) {}
func TestBehavior11(t *testing.T) {}
func TestBehavior12(t *testing.T) {}
func TestBehavior13(t *testing.T) {}
func TestBehavior14(t *testing.T) {}
func TestBehavior15(t *testing.T) {}
func TestBehavior16(t *testing.T) {}
func TestBehavior17(t *testing.T) {}
func TestBehavior18(t *testing.T) {}
func TestBehavior19(t *testing.T) {}
func TestBehavior20(t *testing.T) {}
func TestBehavior21(t *testing.T) {}
func TestBehavior22(t *testing.T) {}
func TestBehavior23(t *testing.T) {}
func TestBehavior24(t *testing.T) {}
func TestBehavior25(t *testing.T) {}
`

	if err := os.WriteFile(filepath.Join(deepcacheDir, "deepcache_test.go"), []byte(testCode), 0644); err != nil {
		t.Fatalf("failed to write deepcache_test.go: %v", err)
	}

	// Create 8 caller packages importing deepcache (Ca=8, Ce=0 → I=0.0)
	for i := 1; i <= 8; i++ {
		callerDir := filepath.Join(tempDir, fmt.Sprintf("caller%d", i))
		os.Mkdir(callerDir, 0755)
		callerCode := fmt.Sprintf(`package caller%d

import "example.com/cache/deepcache"

func Call%d(cfg deepcache.Config) {
	_ = cfg
}
`, i, i)
		if err := os.WriteFile(filepath.Join(callerDir, fmt.Sprintf("caller%d.go", i)), []byte(callerCode), 0644); err != nil {
			t.Fatalf("failed to write caller%d: %v", i, err)
		}
	}

	report, err := CheckWithSurfaceRatio([]string{tempDir}, 0.5, 0.5, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// deepcache should NOT be in violations (test funcs excluded, real ratio stays under gate)
	for i := range report.Violations {
		if report.Violations[i].ImportPath == "example.com/cache/deepcache" {
			t.Fatalf("deepcache should not be flagged (test funcs excluded), got SurfaceRatio=%f (includes TestBehaviorN?)", report.Violations[i].SurfaceRatio)
		}
	}
}
