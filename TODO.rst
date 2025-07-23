=======
TODO List
=======


bug - env exclusion of .gitignore had lower priority than -f
Csense/capability-assessment-suite refactor/m…!?↑4 via 🐳 colima via 🐹 v1.24.0 via 🐍 v3.12.2 with 🅒 base❯ codecat -o framework.txt -d services/framework_knowledge_go -d protos -x protos/gen -d pkg/env -d pkg/canonical_models

--- Summary ---
Included 71 files (439.1 KiB total) relative to CWD 'capability-assessment-suite':
├── pkg
│   └── canonical_models
│       └── framework.go (10.4 KiB)
├── protos
│   ├── assessment.proto (2.6 KiB)
│   ├── charting.proto (2.4 KiB)
│   ├── common.proto (1.1 KiB)
│   ├── framework.proto (9.9 KiB)
│   ├── iam.proto (2.7 KiB)
│   ├── ledger.proto (12.1 KiB)
│   ├── model_gateway.proto (6.2 KiB)
│   ├── orchestration.proto (3.5 KiB)
│   ├── reporting.proto (2.5 KiB)
│   └── text_extraction.proto (2.0 KiB)
└── services
    └── framework_knowledge_go
        ├── NOTES.md (27.0 KiB)
        ├── cmd
        │   └── server
        │       └── main.go (2.7 KiB)
        ├── go.mod (1.1 KiB)
        ├── go.sum (5.2 KiB)
        ├── internal
        │   ├── server
        │   │   ├── agent_handlers.go (2.9 KiB)
        │   │   ├── framework_handlers.go (8.2 KiB)
        │   │   ├── grpc.go (1.3 KiB)
        │   │   ├── ledger_handlers.go (13.9 KiB)
        │   │   └── narrative_handlers.go (2.9 KiB)
        │   └── storage
        │       ├── agent.go (3.5 KiB)
        │       ├── assessment.go (17.8 KiB)
        │       ├── assessment_retrieval.go (8.7 KiB)
        │       ├── audit_retrieval.go (7.0 KiB)
        │       ├── cypher
        │       │   ├── agent.cql (893 B)
        │       │   ├── assessment.cql (7.9 KiB)
        │       │   ├── ingest.cql (2.8 KiB)
        │       │   ├── narratives.cql (4.2 KiB)
        │       │   └── retrieval.cql (4.2 KiB)
        │       ├── framework_retrieval.go (4.1 KiB)
        │       ├── ingest.go (12.9 KiB)
        │       ├── narratives.go (8.0 KiB)
        │       ├── neo4j.go (4.6 KiB)
        │       ├── query_loader.go (2.1 KiB)
        │       ├── reconciliation.go (4.8 KiB)
        │       ├── reconstruction.go (12.8 KiB)
        │       ├── repository.go (2.9 KiB)
        │       ├── scoring.go (7.7 KiB)
        │       └── utils.go (6.3 KiB)
        └── test_client
            ├── check_project_graph.sh (1.2 KiB)
            ├── run_all_tests.sh (5.1 KiB)
            ├── test_assertion_validation.py (6.0 KiB)
            ├── test_assessment_flow.py (7.0 KiB)
            ├── test_audit_and_history.py (9.7 KiB)
            ├── test_cases
            │   ├── assertions
            │   │   ├── conflict_with_unassessed_assertions.json (2.0 KiB)
            │   │   ├── full_shriram_assertions.json (8.5 KiB)
            │   │   ├── hr-4ind_assertions.json (3.7 KiB)
            │   │   ├── hr-4ind_complex_status_assertions.json (2.8 KiB)
            │   │   ├── multi-cap_assertions.json (1.1 KiB)
            │   │   ├── unassessed_status_assertions.json (1.1 KiB)
            │   │   └── v2_conflict_assertions.json (2.6 KiB)
            │   ├── comprehensive
            │   │   ├── test_shriram021.json (5.5 KiB)
            │   │   └── test_waf1.json (1.7 KiB)
            │   ├── hr-4ind.json (10.1 KiB)
            │   ├── multi-cap.json (1.9 KiB)
            │   ├── narratives
            │   │   ├── grouping_narrative.json (769 B)
            │   │   ├── narratives_for_all_entity_types.json (1.7 KiB)
            │   │   └── versioning_and_filtering.json (1.5 KiB)
            │   ├── shriram021.json (18.0 KiB)
            │   └── waf1.json (17.8 KiB)
            ├── test_comprehensive_flow.py (10.9 KiB)
            ├── test_concurrency.py (8.0 KiB)
            ├── test_framework_integrity.py (8.9 KiB)
            ├── test_full_reconciliation.py (7.3 KiB)
            ├── test_grouping_narrative.py (6.8 KiB)
            ├── test_ledger_page_creation.py (6.3 KiB)
            ├── test_narrative_entity_types.py (7.6 KiB)
            ├── test_narrative_isolation.py (7.4 KiB)
            ├── test_narratives.py (10.8 KiB)
            ├── test_reconciliation_consistency.py (6.1 KiB)
            └── test_utils.py (5.4 KiB)

Empty files found (1):
- protos/__init__.py

Errors encountered (0):
---------------
Csense/capability-assessment-suite refactor/m…!?↑4 via 🐳 colima via 🐹 v1.24.0 via 🐍 v3.12.2 with 🅒 base125ms ❯ ls pkg/env              
env.go


High Priority / Next Steps
--------------------------

*   **Implement Metadata Output Options:** Add config options to add empty file listing to output, as this could be important infomation about the codebase (e.g. `__init__.py`` presence).
*   **Fix integration tests:** formatting matching is broken currently.
*   **Fix Unit Tests:** broken currently.

Medium Priority / Refinements
-----------------------------

*   **Refactor Tests:** Ensure all tests are in the most appropriate `*_test.go` file.
*   **Review Usecases:** Is everything covered?
*   **Files with comma in the name problem:** `codecat -e a\,go -f 'b\,c.log'`

Low Priority / Future Ideas
---------------------------

*   **Piping Improvements:** Ensure piping (`codecat | other_cmd`) works flawlessly, potentially adding flags to suppress summary output to stderr if needed. Ref: UC-006.

