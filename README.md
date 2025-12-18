# tnrpt

## Lexer → CST → AST pipeline
Lexer responsibilities:
* Tokens with spaces embedded as leading trivia.
* End of Line and End of File reported as tokens.
* Line, Column, and position/span for all tokens.

CST responsibilities:
* Represent raw grammar structure.
* Nodes carry token slices.
* Syntactic errors emit Diagnostic using CST.Spans.

AST responsibilities:
* Only copy needed information (semantic value (e.g., UnitIDExpr.Text) and Span).
* AST does not refer back to CST nodes.
* Semantic passes emit Diagnostic using AST.Spans.

Result:
* Clean separation
* No redundant source strings
* Excellent error messages
* Easy source reconstruction
* Lightweight tokens
* Easy to write golden tests