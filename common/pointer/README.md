# Pointer Package Conventions

`gopkg/common/pointer` provides shared pointer helpers for Go modules across this monorepo.

This package owns generic pointer mechanics only:
- creating pointers from values
- extracting values from pointers with zero-value fallback
- returning nil for zero-value inputs when callers need optional pointer semantics

This package does not own:
- transport-specific serialization rules
- repository nullability policy
- business-specific defaulting behavior

Keep exported APIs generic and avoid adding helpers that only make sense for one adapter or feature.
