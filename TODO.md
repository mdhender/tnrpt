# Implementation

Stages
* [ ] Stage to import turn report (`.docx`) file to database
* [ ] Stage to import turn report (`.report.txt`) file to database
* [ ] Stage to split turn report into sections
* [ ] Stage to parse turn report section
* [ ] Stage to walk parsed section into unit/tile observations
* [ ] Stage to render unit/tile observations into Worldographer maps

## Sprint 10

The goal of Sprint 10 is to implement the `bistre` parser.
The parser is a copy of `azul` saved to the pipelines/parsers/bistre folder.

### Plan 
* [x] Create pipelines/parsers/bistre folder
* [x] Copy `azul` code into `bistre`
* [x] Update `cmd/tnrpt` pipeline to use `bistre` instead of `azul`.
