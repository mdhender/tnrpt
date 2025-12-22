package model

import (
	"time"

	"github.com/maloquacious/hexg"
)

// ReportFile is the original report document sent by the GM (DOCX/PDF/etc).
type ReportFile struct {
	ID        int64     `json:"id"        db:"id"`
	Game      string    `json:"game"      db:"game"`
	ClanNo    string    `json:"clanNo"    db:"clan_no"`
	TurnNo    int       `json:"turnNo"    db:"turn_no"`
	Name      string    `json:"name"      db:"name"` // original filename (untainted or sanitized)
	SHA256    string    `json:"sha256"    db:"sha256"`
	Mime      string    `json:"mime"      db:"mime"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// ReportX is the extracted subset of a ReportFile (map-render-relevant only).
type ReportX struct {
	ID           int64     `json:"id"           db:"id"`
	ReportFileID int64     `json:"reportFileId" db:"report_file_id"`
	Game         string    `json:"game"         db:"game"`
	ClanNo       string    `json:"clanNo"       db:"clan_no"`
	TurnNo       int       `json:"turnNo"       db:"turn_no"`
	CreatedAt    time.Time `json:"createdAt"    db:"created_at"`
	Units        []*UnitX  `json:"units,omitempty"` // for JSON export/import
}

// UnitX is one unit section in a report extract.
type UnitX struct {
	// Natural key (for import/merge): (report_x_id, unit_id)
	ID        int64  `json:"id"        db:"id"`
	ReportXID int64  `json:"reportXId" db:"report_x_id"`
	UnitID    string `json:"unitId"    db:"unit_id"` // e.g., "0987c4"

	TurnNo int `json:"turnNo" db:"turn_no"` // e.g., 90304

	StartTN TNCoord `json:"startTN" db:"-"` // e.g., "QQ 0205"
	EndTN   TNCoord `json:"endTN"   db:"-"` // e.g., "QQ 0205"

	Acts []*Act  `json:"acts,omitempty"`        // ordered list for JSON export/import
	Src  *SrcRef `json:"src,omitempty" db:"-"` // provenance (optional but recommended)
}

// TNCoord is a TribeNet coordinate (as seen in reports).
type TNCoord string // e.g., "QQ 0205"

// ActKind discriminates Act variants.
type ActKind string

const (
	ActKindFollow ActKind = "follow"
	ActKindGoto   ActKind = "goto"
	ActKindMove   ActKind = "move"
	ActKindScout  ActKind = "scout"
	ActKindStatus ActKind = "status"
)

// StepKind discriminates Step variants.
type StepKind string

const (
	StepKindAdv    StepKind = "adv"
	StepKindStill  StepKind = "still"
	StepKindPatrol StepKind = "patrol"
	StepKindObs    StepKind = "obs"
)

// SrcRef is optional provenance used to resolve merge conflicts.
// Not directly persisted as a row; repository layer flattens into parent table columns.
type SrcRef struct {
	DocID   int64  `json:"docId"`            // report_file_id (or other document id)
	UnitID  string `json:"unitId,omitempty"` // originating unit
	TurnNo  int    `json:"turnNo,omitempty"`
	ActSeq  int    `json:"actSeq,omitempty"`  // sequence in UnitX.Acts (1-based)
	StepSeq int    `json:"stepSeq,omitempty"` // sequence in Act.Steps (1-based)
	Note    string `json:"note,omitempty"`
}

// Act is an action in an extracted unit section.
// Kind discriminator: follow | goto | move | scout | status
//
// To avoid polymorphic pain in SQLite, this struct keeps a small set of
// kind-specific fields that map to nullable columns in the `acts` table,
// while Steps (for move/scout/status) live in `steps`.
type Act struct {
	ID      int64   `json:"id"             db:"id"`
	UnitXID int64   `json:"unitXId"        db:"unit_x_id"`
	Seq     int     `json:"seq"            db:"seq"`  // ordering within unit section (1-based)
	Kind    ActKind `json:"kind"           db:"kind"` // follow|goto|move|scout|status
	Ok      bool   `json:"ok,omitempty"   db:"ok"`   // coarse result at action level
	Note    string `json:"note,omitempty" db:"note"`

	// follow
	TargetUnitID string `json:"targetUnitId,omitempty" db:"target_unit_id"`

	// goto
	DestTN TNCoord `json:"destTN,omitempty" db:"-"` // e.g., "QQ 1010"

	// move/scout/status steps (status will generally have 1 obs step)
	Steps []*Step `json:"steps,omitempty"`

	Src *SrcRef `json:"src,omitempty" db:"-"` // provenance (optional but recommended)
}

// Step is an atomic step within move/scout/status.
// Kind discriminator: adv | still | patrol | obs
//
// Encounters, terrain, and borders are normalized in child tables keyed by step_id.
type Step struct {
	ID    int64    `json:"id"    db:"id"`
	ActID int64    `json:"actId" db:"act_id"`
	Seq   int      `json:"seq"   db:"seq"`  // 1-based
	Kind  StepKind `json:"kind"  db:"kind"` // adv|still|patrol|obs

	// common result
	Ok   bool   `json:"ok,omitempty"   db:"ok"`
	Note string `json:"note,omitempty" db:"note"`

	// adv payload
	Dir     string `json:"dir,omitempty"     db:"dir"`      // e.g. N,NE,SE,S,SW,NW
	FailWhy string `json:"failWhy,omitempty" db:"fail_why"` // terrain|exhaust|blocked|unknown|...

	// obs payload (flattened bits; details in child tables)
	Terr    string `json:"terr,omitempty"    db:"terr"`    // terrain code/name
	Special bool   `json:"special,omitempty" db:"special"` // special hex flag
	Label   string `json:"label,omitempty"   db:"label"`   // label if special

	// Embedded for JSON convenience; stored normalized in child tables.
	Enc     *Enc         `json:"enc,omitempty"     db:"-"`
	Borders []*BorderObs `json:"borders,omitempty" db:"-"`

	Src *SrcRef `json:"src,omitempty" db:"-"` // provenance (optional but recommended)
}

// Enc is a reusable encounter container used by patrol/obs steps and optionally tiles.
type Enc struct {
	Units []*UnitSeen   `json:"units,omitempty"`
	Sets  []*SettleSeen `json:"sets,omitempty"`
	Rsrc  []*RsrcSeen   `json:"rsrc,omitempty"`
}

// UnitSeen is a unit encounter.
type UnitSeen struct {
	UnitID string `json:"unitId"           db:"unit_id"` // e.g., "9123g1"
	Name   string `json:"name,omitempty"   db:"name"`
	ClanNo string `json:"clanNo,omitempty" db:"clan_no"`
}

// SettleSeen is a settlement encounter.
type SettleSeen struct {
	Name   string `json:"name"             db:"name"`
	Kind   string `json:"kind,omitempty"   db:"kind"` // city/fort/ruin/...
	ClanNo string `json:"clanNo,omitempty" db:"clan_no"`
}

// RsrcSeen is a resource encounter.
type RsrcSeen struct {
	Kind string `json:"kind"          db:"kind"` // ore/food/herb/...
	Qty  int    `json:"qty,omitempty" db:"qty"`
}

// BorderObs is a border observation, usually from obs steps and tiles.
type BorderObs struct {
	Dir  string `json:"dir"  db:"dir"`
	Kind string `json:"kind" db:"kind"` // river/ford/road/cliff/...
}

// Tile is the walker output: observed state at a hex coordinate.
// Organized by coordinates and mergeable across sources.
type Tile struct {
	ID  int64    `json:"id"  db:"id"`
	Hex hexg.Hex `json:"hex" db:"hex"`

	Terr         string        `json:"terr,omitempty"         db:"terr"`
	SpecialLabel string        `json:"specialLabel,omitempty" db:"special_label"`
	Units        []*UnitSeen   `json:"units,omitempty"        db:"-"`
	Sets         []*SettleSeen `json:"sets,omitempty"         db:"-"`
	Rsrc         []*RsrcSeen   `json:"rsrc,omitempty"         db:"-"`
	Borders      []*BorderObs  `json:"borders,omitempty"      db:"-"`

	// Provenance for merge conflicts: which extracted records contributed.
	Src []*TileSrc `json:"src,omitempty" db:"-"`
}

// TileSrc is provenance for tiles; it lets you trace/resolve merge conflicts.
type TileSrc struct {
	DocID   int64  `json:"docId"             db:"doc_id"`
	UnitID  string `json:"unitId,omitempty"  db:"unit_id"`
	TurnNo  int    `json:"turnNo,omitempty"  db:"turn_no"`
	ActSeq  int    `json:"actSeq,omitempty"  db:"act_seq"`  // 1-based
	StepSeq int    `json:"stepSeq,omitempty" db:"step_seq"` // 1-based
	Note    string `json:"note,omitempty"    db:"note"`
}

// RenderJob describes a render request (units + turns + params).
type RenderJob struct {
	ID        int64     `json:"id"        db:"id"`
	Game      string    `json:"game"      db:"game"`
	ClanNo    string    `json:"clanNo"    db:"clan_no"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// Inputs
	UnitIDs []string `json:"unitIds,omitempty" db:"-"` // stored in render_job_units
	Turns   []int    `json:"turns,omitempty"   db:"-"` // stored in render_job_turns

	// Output metadata (if persisted)
	WxxPath string `json:"wxxPath,omitempty" db:"wxx_path"`
	WxxSHA  string `json:"wxxSha,omitempty"  db:"wxx_sha"`
}
