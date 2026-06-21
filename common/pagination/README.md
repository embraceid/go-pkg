# Pagination Package Conventions

`gopkg/common/pagination` provides shared pagination primitives for Go modules across this monorepo.

This package owns generic pagination mechanics only:
- default page and limit values
- pagination state such as page, limit, total, and total page
- offset calculation and page increment helpers
- total-page calculation from record counts

This package does not own:
- transport-specific query parsing
- repository filtering semantics
- response serialization policy
- business-specific paging rules

Keep exported APIs generic and avoid adding helpers that only make sense for one adapter or feature.
