-- SQLite schema for OttoMap agent-optimized model
-- Coordinates are stored flattened for DB writes; JSON uses TNCoord/HexCoord objects.
PRAGMA foreign_keys = ON;

-- Upload batches: groups multiple files in one ingest operation
CREATE TABLE IF NOT EXISTS upload_batches (
                                              id         INTEGER PRIMARY KEY,
                                              game       TEXT    NOT NULL,
                                              clan_no    TEXT    NOT NULL,
                                              turn_no    INTEGER NOT NULL,
                                              created_by TEXT,                -- CLI user or web session
                                              created_at TEXT    NOT NULL     -- ISO8601 UTC
);
CREATE INDEX IF NOT EXISTS idx_upload_batches_game_turn
    ON upload_batches(game, turn_no, clan_no);

-- Source documents
CREATE TABLE IF NOT EXISTS report_files (
                                            id          INTEGER PRIMARY KEY,
                                            game        TEXT NOT NULL,
                                            clan_no     TEXT NOT NULL,
                                            turn_no     INTEGER NOT NULL,
                                            name        TEXT NOT NULL,
                                            sha256      TEXT NOT NULL,
                                            mime        TEXT NOT NULL,
                                            created_at  TEXT NOT NULL,
                                            fs_path     TEXT,     -- Relative to data-dir; e.g., "batches/1/0512.docx"
                                            batch_id    INTEGER REFERENCES upload_batches(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_report_files_game_turn_clan ON report_files(game, turn_no, clan_no);
CREATE INDEX IF NOT EXISTS idx_report_files_sha256 ON report_files(sha256);
CREATE INDEX IF NOT EXISTS idx_report_files_batch ON report_files(batch_id);

-- Extract roots
CREATE TABLE IF NOT EXISTS report_extracts (
                                               id             INTEGER PRIMARY KEY,
                                               report_file_id INTEGER NOT NULL REFERENCES report_files(id) ON DELETE CASCADE,
                                               game           TEXT NOT NULL,
                                               clan_no        TEXT NOT NULL,
                                               turn_no        INTEGER NOT NULL,
                                               created_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_report_extracts_report_file_id ON report_extracts(report_file_id);
CREATE INDEX IF NOT EXISTS idx_report_extracts_game_turn_clan ON report_extracts(game, turn_no, clan_no);

-- One row per unit section in an extract
CREATE TABLE IF NOT EXISTS unit_extracts (
                                             id           INTEGER PRIMARY KEY,
                                             report_x_id  INTEGER NOT NULL REFERENCES report_extracts(id) ON DELETE CASCADE,
                                             unit_id      TEXT NOT NULL,
                                             clan_id      TEXT NOT NULL,  -- owning clan (e.g., "500" extracted from unit_id)
                                             turn_no      INTEGER NOT NULL,

                                             start_grid   TEXT NOT NULL,
                                             start_col    INTEGER NOT NULL,
                                             start_row    INTEGER NOT NULL,

                                             end_grid     TEXT NOT NULL,
                                             end_col      INTEGER NOT NULL,
                                             end_row      INTEGER NOT NULL,

    -- provenance (optional; helps with debugging/merges)
                                             src_doc_id   INTEGER,
                                             src_note     TEXT,

                                             UNIQUE(report_x_id, unit_id)
);
CREATE INDEX IF NOT EXISTS idx_unit_extracts_report_x ON unit_extracts(report_x_id);
CREATE INDEX IF NOT EXISTS idx_unit_extracts_clan ON unit_extracts(clan_id);

-- Acts: single table w/ kind discriminator and nullable kind-specific columns
CREATE TABLE IF NOT EXISTS acts (
                                    id            INTEGER PRIMARY KEY,
                                    unit_x_id     INTEGER NOT NULL REFERENCES unit_extracts(id) ON DELETE CASCADE,
                                    seq           INTEGER NOT NULL,
                                    kind          TEXT NOT NULL, -- follow|goto|move|scout|status
                                    ok            INTEGER,       -- NULL/0/1
                                    note          TEXT,

    -- follow payload
                                    target_unit_id TEXT,

    -- goto payload
                                    dest_grid     TEXT,
                                    dest_col      INTEGER,
                                    dest_row      INTEGER,

    -- provenance (optional)
                                    src_doc_id    INTEGER,
                                    src_turn_no   INTEGER,
                                    src_unit_id   TEXT,
                                    src_act_seq   INTEGER,
                                    src_note      TEXT,

                                    UNIQUE(unit_x_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_acts_unit_x ON acts(unit_x_id);

-- Steps: single table w/ kind discriminator and nullable kind-specific columns
CREATE TABLE IF NOT EXISTS steps (
                                     id        INTEGER PRIMARY KEY,
                                     act_id    INTEGER NOT NULL REFERENCES acts(id) ON DELETE CASCADE,
                                     seq       INTEGER NOT NULL,
                                     kind      TEXT NOT NULL, -- adv|still|patrol|obs
                                     ok        INTEGER,       -- NULL/0/1
                                     note      TEXT,

    -- adv payload
                                     dir       TEXT,
                                     fail_why  TEXT,

    -- obs payload (flattened; details in child tables)
                                     terr      TEXT,
                                     special   INTEGER NOT NULL DEFAULT 0,
                                     label     TEXT,

    -- provenance (optional)
                                     src_doc_id   INTEGER,
                                     src_turn_no  INTEGER,
                                     src_unit_id  TEXT,
                                     src_act_seq  INTEGER,
                                     src_step_seq INTEGER,
                                     src_note     TEXT,

                                     UNIQUE(act_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_steps_act ON steps(act_id);
CREATE INDEX IF NOT EXISTS idx_steps_kind ON steps(kind);

-- Encounters normalized by step_id
CREATE TABLE IF NOT EXISTS step_enc_units (
                                              id      INTEGER PRIMARY KEY,
                                              step_id INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
                                              unit_id TEXT NOT NULL,
                                              name    TEXT,
                                              clan_no TEXT
);
CREATE INDEX IF NOT EXISTS idx_step_enc_units_step ON step_enc_units(step_id);

CREATE TABLE IF NOT EXISTS step_enc_sets (
                                             id      INTEGER PRIMARY KEY,
                                             step_id INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
                                             name    TEXT NOT NULL,
                                             kind    TEXT,
                                             clan_no TEXT
);
CREATE INDEX IF NOT EXISTS idx_step_enc_sets_step ON step_enc_sets(step_id);

CREATE TABLE IF NOT EXISTS step_enc_rsrc (
                                             id      INTEGER PRIMARY KEY,
                                             step_id INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
                                             kind    TEXT NOT NULL,
                                             qty     INTEGER
);
CREATE INDEX IF NOT EXISTS idx_step_enc_rsrc_step ON step_enc_rsrc(step_id);

-- Borders normalized by step_id
CREATE TABLE IF NOT EXISTS step_borders (
                                            id      INTEGER PRIMARY KEY,
                                            step_id INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
                                            dir     TEXT NOT NULL,
                                            kind    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_step_borders_step ON step_borders(step_id);

-- Walker output: tiles keyed by hex coordinate
CREATE TABLE IF NOT EXISTS tiles (
                                     id            INTEGER PRIMARY KEY,
                                     hex           TEXT NOT NULL, -- hexg.Hex.ConciseString() format
                                     terr          TEXT,
                                     special_label TEXT,
                                     UNIQUE(hex)
);
CREATE INDEX IF NOT EXISTS idx_tiles_hex ON tiles(hex);

-- Tile contents (denormalized lists)
CREATE TABLE IF NOT EXISTS tile_units (
                                          id      INTEGER PRIMARY KEY,
                                          tile_id INTEGER NOT NULL REFERENCES tiles(id) ON DELETE CASCADE,
                                          unit_id TEXT NOT NULL,
                                          name    TEXT,
                                          clan_no TEXT
);
CREATE INDEX IF NOT EXISTS idx_tile_units_tile ON tile_units(tile_id);

CREATE TABLE IF NOT EXISTS tile_sets (
                                         id      INTEGER PRIMARY KEY,
                                         tile_id INTEGER NOT NULL REFERENCES tiles(id) ON DELETE CASCADE,
                                         name    TEXT NOT NULL,
                                         kind    TEXT,
                                         clan_no TEXT
);
CREATE INDEX IF NOT EXISTS idx_tile_sets_tile ON tile_sets(tile_id);

CREATE TABLE IF NOT EXISTS tile_rsrc (
                                         id      INTEGER PRIMARY KEY,
                                         tile_id INTEGER NOT NULL REFERENCES tiles(id) ON DELETE CASCADE,
                                         kind    TEXT NOT NULL,
                                         qty     INTEGER
);
CREATE INDEX IF NOT EXISTS idx_tile_rsrc_tile ON tile_rsrc(tile_id);

CREATE TABLE IF NOT EXISTS tile_borders (
                                            id      INTEGER PRIMARY KEY,
                                            tile_id INTEGER NOT NULL REFERENCES tiles(id) ON DELETE CASCADE,
                                            dir     TEXT NOT NULL,
                                            kind    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tile_borders_tile ON tile_borders(tile_id);

--  Copyright (c) 2025 Michael D Henderson. All rights reserved.

-- Tile provenance for merge conflicts
CREATE TABLE IF NOT EXISTS tile_src (
                                        id       INTEGER PRIMARY KEY,
                                        tile_id  INTEGER NOT NULL REFERENCES tiles(id) ON DELETE CASCADE,
                                        doc_id   INTEGER NOT NULL,
                                        unit_id  TEXT,
                                        turn_no  INTEGER,
                                        act_seq  INTEGER,
                                        step_seq INTEGER,
                                        note     TEXT
);
CREATE INDEX IF NOT EXISTS idx_tile_src_tile ON tile_src(tile_id);
CREATE INDEX IF NOT EXISTS idx_tile_src_doc ON tile_src(doc_id);

-- Render jobs (optional persistence)
CREATE TABLE IF NOT EXISTS render_jobs (
                                           id         INTEGER PRIMARY KEY,
                                           game       TEXT NOT NULL,
                                           clan_no    TEXT NOT NULL,
                                           created_at TEXT NOT NULL,
                                           wxx_path   TEXT,
                                           wxx_sha    TEXT
);
CREATE INDEX IF NOT EXISTS idx_render_jobs_game_clan ON render_jobs(game, clan_no);

CREATE TABLE IF NOT EXISTS render_job_units (
                                                id        INTEGER PRIMARY KEY,
                                                job_id    INTEGER NOT NULL REFERENCES render_jobs(id) ON DELETE CASCADE,
                                                unit_id   TEXT NOT NULL,
                                                UNIQUE(job_id, unit_id)
);

CREATE TABLE IF NOT EXISTS render_job_turns (
                                                id        INTEGER PRIMARY KEY,
                                                job_id    INTEGER NOT NULL REFERENCES render_jobs(id) ON DELETE CASCADE,
                                                turn_no   INTEGER NOT NULL,
                                                UNIQUE(job_id, turn_no)
);

-- Users and authentication
CREATE TABLE IF NOT EXISTS users (
                                     handle        TEXT PRIMARY KEY,
                                     user_name     TEXT NOT NULL,
                                     email         TEXT,
                                     timezone      TEXT,
                                     password_hash TEXT NOT NULL,
                                     created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_roles (
                                          id          INTEGER PRIMARY KEY,
                                          user_handle TEXT NOT NULL REFERENCES users(handle) ON DELETE CASCADE,
                                          role        TEXT NOT NULL,
                                          UNIQUE(user_handle, role)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_handle ON user_roles(user_handle);

-- Games and clan membership (clan_no is per-game, not per-user)
CREATE TABLE IF NOT EXISTS games (
                                     id          TEXT PRIMARY KEY,
                                     description TEXT
);

CREATE TABLE IF NOT EXISTS game_clans (
                                          id          INTEGER PRIMARY KEY,
                                          game_id     TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
                                          user_handle TEXT NOT NULL REFERENCES users(handle) ON DELETE CASCADE,
                                          clan_no     INTEGER NOT NULL,
                                          UNIQUE(game_id, user_handle),
                                          UNIQUE(game_id, clan_no)
);
CREATE INDEX IF NOT EXISTS idx_game_clans_game ON game_clans(game_id);
CREATE INDEX IF NOT EXISTS idx_game_clans_user ON game_clans(user_handle);

--  Copyright (c) 2025 Michael D Henderson. All rights reserved.

-- Game turns (year/month, is_active, due_date in UTC)
CREATE TABLE IF NOT EXISTS game_turns (
                                          id          INTEGER PRIMARY KEY,
                                          game_id     TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
                                          turn_id     INTEGER NOT NULL,  -- e.g., 89912 for year 899 month 12
                                          year        INTEGER NOT NULL,
                                          month       INTEGER NOT NULL,
                                          is_active   INTEGER NOT NULL DEFAULT 0,
                                          due_date    TEXT,  -- ISO8601 UTC timestamp
                                          UNIQUE(game_id, turn_id)
);
CREATE INDEX IF NOT EXISTS idx_game_turns_game ON game_turns(game_id);
CREATE INDEX IF NOT EXISTS idx_game_turns_active ON game_turns(game_id, is_active);

-- Work queue for pipeline stages
CREATE TABLE IF NOT EXISTS work (
                                    id             INTEGER PRIMARY KEY,
                                    report_file_id INTEGER NOT NULL REFERENCES report_files(id) ON DELETE CASCADE,

                                    stage          TEXT    NOT NULL,                  -- 'extract', 'parse'
                                    status         TEXT    NOT NULL DEFAULT 'queued', -- queued|running|ok|failed

                                    attempt        INTEGER NOT NULL DEFAULT 0,
                                    available_at   TEXT    NOT NULL,                  -- ISO8601 UTC
                                    locked_by      TEXT,                              -- worker ID
                                    locked_at      TEXT,                              -- ISO8601 UTC
                                    started_at     TEXT,                              -- first execution time
                                    finished_at    TEXT,                              -- ISO8601 UTC

                                    error_code     TEXT,                              -- e.g., "PARSE_SYNTAX_ERROR"
                                    error_message  TEXT,

                                    UNIQUE(report_file_id, stage)
);
CREATE INDEX IF NOT EXISTS idx_work_ready ON work(status, stage, available_at);
CREATE INDEX IF NOT EXISTS idx_work_file ON work(report_file_id);
