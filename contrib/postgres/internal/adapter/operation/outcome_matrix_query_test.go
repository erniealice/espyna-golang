//go:build postgresql

package operation

import (
	"database/sql"
	"testing"
)

// TestComposePhaseLabel pins the phase-header parenthetical (S8 §3): a phase with
// a sub-deliverable variant renders "NAME (VARIANT_NAME)"; a phase with no variant
// (NULL / blank) renders the bare NAME. Single composition site (adapter).
func TestComposePhaseLabel(t *testing.T) {
	cases := []struct {
		name    string
		phase   string
		variant sql.NullString
		want    string
	}{
		{"no_variant_null", "Semester 1", sql.NullString{Valid: false}, "Semester 1"},
		{"no_variant_blank", "Semester 1", sql.NullString{String: "", Valid: true}, "Semester 1"},
		{"no_variant_whitespace", "Semester 1", sql.NullString{String: "   ", Valid: true}, "Semester 1"},
		{"variant_visual_arts", "Semester 1", sql.NullString{String: "Visual Arts", Valid: true}, "Semester 1 (Visual Arts)"},
		{"variant_music", "Semester 2", sql.NullString{String: "Music", Valid: true}, "Semester 2 (Music)"},
		{"variant_trimmed", "Semester 2", sql.NullString{String: "  TLE  ", Valid: true}, "Semester 2 (TLE)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := composePhaseLabel(c.phase, c.variant); got != c.want {
				t.Errorf("composePhaseLabel(%q, %+v) = %q, want %q", c.phase, c.variant, got, c.want)
			}
		})
	}
}

// TestComputeCellEditable pins the editable rule, including the S8 §E empty-cell
// guard: an EMPTY cell is editable only by the staff the task is ASSIGNED to, so a
// merged two-teacher class cannot leak cross-strand empty-cell edits; RECORDED
// cells keep recorder-only semantics unchanged; non-staff never edit.
func TestComputeCellEditable(t *testing.T) {
	const me, other = "staff-me", "staff-other"

	cases := []struct {
		name        string
		hasOutcome  bool
		staffOK     bool
		recordedBy  string
		assignedTo  string
		actingStaff string
		jobTaskID   string
		want        bool
	}{
		// --- empty cells (S8 §E guard) ---
		{"empty_assigned_to_me_editable", false, true, "", me, me, "jt-1", true},
		{"empty_assigned_to_other_NOT_editable", false, true, "", other, me, "jt-1", false},
		{"empty_unassigned_NOT_editable", false, true, "", "", me, "jt-1", false},
		{"empty_no_instance_NOT_editable", false, true, "", me, me, "", false},
		{"empty_non_staff_NOT_editable", false, false, "", me, me, "jt-1", false},
		// --- recorded cells (semantics unchanged: recorder-only) ---
		{"recorded_by_me_editable", true, true, me, other, me, "jt-1", true},
		{"recorded_by_other_NOT_editable", true, true, other, me, me, "jt-1", false},
		{"recorded_non_staff_NOT_editable", true, false, me, me, me, "jt-1", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeCellEditable(c.hasOutcome, c.staffOK, c.recordedBy, c.assignedTo, c.actingStaff, c.jobTaskID)
			if got != c.want {
				t.Errorf("computeCellEditable(hasOutcome=%v,staffOK=%v,recordedBy=%q,assignedTo=%q,actingStaff=%q,jobTaskID=%q) = %v, want %v",
					c.hasOutcome, c.staffOK, c.recordedBy, c.assignedTo, c.actingStaff, c.jobTaskID, got, c.want)
			}
		})
	}
}
