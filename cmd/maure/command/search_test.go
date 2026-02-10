package command

import (
	"testing"

	"maure/pkg/index"
)

func TestSearchCommandFlagsParseFromSize(t *testing.T) {
	cmd := NewSearchCommand()
	if err := cmd.Flags().Parse([]string{"--from", "3", "--size", "5", "hello"}); err != nil {
		t.Fatalf("parse flags failed: %v", err)
	}

	if cmd.from != 3 {
		t.Fatalf("expected from=3, got %d", cmd.from)
	}
	if cmd.size != 5 {
		t.Fatalf("expected size=5, got %d", cmd.size)
	}
	if cmd.legacyNUsed {
		t.Fatalf("expected legacy n flag false")
	}
}

func TestSearchCommandLegacyNCompatibility(t *testing.T) {
	cmd := NewSearchCommand()
	if err := cmd.Flags().Parse([]string{"-n", "7", "hello"}); err != nil {
		t.Fatalf("parse flags failed: %v", err)
	}

	if cmd.size != 7 {
		t.Fatalf("expected size mapped from -n to be 7, got %d", cmd.size)
	}
	if !cmd.legacyNUsed {
		t.Fatalf("expected legacy n flag to be marked as used")
	}
}

func TestPaginateScoreDocsBoundaries(t *testing.T) {
	in := []index.ScoreDoc{
		{DocID: 1, Score: 1},
		{DocID: 2, Score: 1},
		{DocID: 3, Score: 1},
	}

	page := paginateScoreDocs(in, 1, 2)
	if len(page) != 2 || page[0].DocID != 2 || page[1].DocID != 3 {
		t.Fatalf("unexpected page result: %+v", page)
	}

	page = paginateScoreDocs(in, 3, 2)
	if len(page) != 0 {
		t.Fatalf("expected empty page when from==len, got %d", len(page))
	}
}
