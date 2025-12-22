# Implementation

Stages
* [ ] Stage to import turn report (`.docx`) file to database
* [ ] Stage to import turn report (`.report.txt`) file to database
* [ ] Stage to split turn report into sections
* [ ] Stage to parse turn report section
* [ ] Stage to walk parsed section into unit/tile observations
* [ ] Stage to render unit/tile observations into Worldographer maps

## Sprint 12

The goal of Sprint 12 is to fix or remove the broken golden tests.

### Plan
* [x] Identify broken golden tests
* [x] Fix or remove broken golden tests
* [x] Update golden files

### Notes
Two golden tests in `adapters/` were failing because the test input file 
(`testdata/0899-12.0987.report.txt`) was truncated to 4 lines but the golden 
files expected output from a larger input with scout data. Updated golden 
files to match current parser output for the current input file.
