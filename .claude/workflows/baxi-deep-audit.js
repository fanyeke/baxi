export const meta = {
  name: 'baxi-deep-audit',
  description: 'Deep exploration and comprehensive audit of the Baxi e-commerce governance platform',
  phases: [
    { title: 'Explore', detail: 'Deep codebase exploration across all subsystems' },
    { title: 'Test', detail: 'Run all test suites and measure coverage' },
    { title: 'Analyze', detail: 'Synthesize findings into status report and optimization plan' },
  ],
}

// === Phase 1: Parallel deep exploration across all major subsystems ===
phase('Explore')

const EXPLORATION_TARGETS = [
  {
    label: 'API Layer',
    prompt: `Explore the Baxi Go API layer at /home/zzz/project/baxi/internal/api/ and /home/zzz/project/baxi/cmd/baxi-api/. Analyze:
1. All HTTP handlers, routes, middleware
2. Request/response patterns, error handling
3. Authentication and authorization
4. Input validation
5. Any code smells, missing error handling, or security issues
Report structured findings with file paths and line numbers.`,
    schema: {
      type: 'object',
      properties: {
        subsystem: { type: 'string' },
        files_analyzed: { type: 'number' },
        total_lines: { type: 'number' },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              category: { type: 'string', enum: ['bug', 'security', 'performance', 'code_smell', 'missing_feature'] },
              severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
              title: { type: 'string' },
              description: { type: 'string' },
              file: { type: 'string' },
              line: { type: 'number' },
              recommendation: { type: 'string' }
            },
            required: ['category', 'severity', 'title', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['subsystem', 'findings', 'summary']
    }
  },
  {
    label: 'Decision Engine',
    prompt: `Explore the Baxi decision engine at /home/zzz/project/baxi/internal/decision/ and /home/zzz/project/baxi/internal/service/decision_service.go. Analyze:
1. Decision engine architecture (context builder, case engine, lineage)
2. LLM integration patterns
3. State machine / decision flow
4. Error handling and edge cases
5. Any architectural concerns or improvement opportunities
Report structured findings with file paths and line numbers.`,
    schema: {
      type: 'object',
      properties: {
        subsystem: { type: 'string' },
        files_analyzed: { type: 'number' },
        total_lines: { type: 'number' },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              category: { type: 'string', enum: ['bug', 'security', 'performance', 'code_smell', 'missing_feature'] },
              severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
              title: { type: 'string' },
              description: { type: 'string' },
              file: { type: 'string' },
              line: { type: 'number' },
              recommendation: { type: 'string' }
            },
            required: ['category', 'severity', 'title', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['subsystem', 'findings', 'summary']
    }
  },
  {
    label: 'Data Pipeline',
    prompt: `Explore the Baxi data pipeline at /home/zzz/project/baxi/internal/pipeline/, /home/zzz/project/baxi/internal/ingest/, /home/zzz/project/baxi/internal/alert/, and /home/zzz/project/baxi/internal/recommendation/. Analyze:
1. Pipeline step architecture and orchestration
2. Data flow from ingest to DWD to metrics to anomaly detection to recommendations
3. Error handling, retry logic, idempotency
4. Performance concerns with large datasets
5. Missing validations or edge cases
Report structured findings with file paths and line numbers.`,
    schema: {
      type: 'object',
      properties: {
        subsystem: { type: 'string' },
        files_analyzed: { type: 'number' },
        total_lines: { type: 'number' },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              category: { type: 'string', enum: ['bug', 'security', 'performance', 'code_smell', 'missing_feature'] },
              severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
              title: { type: 'string' },
              description: { type: 'string' },
              file: { type: 'string' },
              line: { type: 'number' },
              recommendation: { type: 'string' }
            },
            required: ['category', 'severity', 'title', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['subsystem', 'findings', 'summary']
    }
  },
  {
    label: 'Governance & Repository',
    prompt: `Explore the Baxi governance and data access layers at /home/zzz/project/baxi/internal/governance/, /home/zzz/project/baxi/internal/repository/, /home/zzz/project/baxi/internal/ontology/, /home/zzz/project/baxi/internal/configloader/. Analyze:
1. Governance model (access policies, data classification, lineage)
2. Repository pattern implementation and DB query safety
3. Ontology and config loading patterns
4. SQL injection risks, missing transaction boundaries
5. Test coverage gaps
Report structured findings with file paths and line numbers.`,
    schema: {
      type: 'object',
      properties: {
        subsystem: { type: 'string' },
        files_analyzed: { type: 'number' },
        total_lines: { type: 'number' },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              category: { type: 'string', enum: ['bug', 'security', 'performance', 'code_smell', 'missing_feature'] },
              severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
              title: { type: 'string' },
              description: { type: 'string' },
              file: { type: 'string' },
              line: { type: 'number' },
              recommendation: { type: 'string' }
            },
            required: ['category', 'severity', 'title', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['subsystem', 'findings', 'summary']
    }
  },
  {
    label: 'Worker & Actions',
    prompt: `Explore the Baxi worker, action, adapter, and outbox systems at /home/zzz/project/baxi/internal/worker/, /home/zzz/project/baxi/internal/action/, /home/zzz/project/baxi/internal/adapter/, /home/zzz/project/baxi/internal/outbox/, /home/zzz/project/baxi/internal/review/. Analyze:
1. Worker dispatch architecture and concurrency patterns
2. Action registry and execution
3. Adapter pattern (Feishu, GitHub, CLI, Manual)
4. Outbox pattern for reliable event delivery
5. Race conditions, deadlocks, error handling
Report structured findings with file paths and line numbers.`,
    schema: {
      type: 'object',
      properties: {
        subsystem: { type: 'string' },
        files_analyzed: { type: 'number' },
        total_lines: { type: 'number' },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              category: { type: 'string', enum: ['bug', 'security', 'performance', 'code_smell', 'missing_feature'] },
              severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
              title: { type: 'string' },
              description: { type: 'string' },
              file: { type: 'string' },
              line: { type: 'number' },
              recommendation: { type: 'string' }
            },
            required: ['category', 'severity', 'title', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['subsystem', 'findings', 'summary']
    }
  },
  {
    label: 'Frontend & MCP',
    prompt: `Explore the Baxi frontend at /home/zzz/project/baxi/frontend/src/ and MCP server at /home/zzz/project/baxi/internal/mcp/ and /home/zzz/project/baxi/cmd/baxi-mcp/. Analyze:
1. React component architecture, routing, state management
2. API client patterns, error handling, loading states
3. Test coverage and quality of existing frontend tests
4. MCP server tools and their implementation
5. UI/UX concerns, missing pages, accessibility
Report structured findings with file paths and line numbers.`,
    schema: {
      type: 'object',
      properties: {
        subsystem: { type: 'string' },
        files_analyzed: { type: 'number' },
        total_lines: { type: 'number' },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              category: { type: 'string', enum: ['bug', 'security', 'performance', 'code_smell', 'missing_feature'] },
              severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
              title: { type: 'string' },
              description: { type: 'string' },
              file: { type: 'string' },
              line: { type: 'number' },
              recommendation: { type: 'string' }
            },
            required: ['category', 'severity', 'title', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['subsystem', 'findings', 'summary']
    }
  },
  {
    label: 'Security & Config',
    prompt: `Perform a security and configuration audit of the Baxi project at /home/zzz/project/baxi/. Analyze:
1. Check .env and .env.example for hardcoded secrets or leaked credentials
2. Review config/*.yml for sensitive data exposure
3. Check authentication middleware, CORS, CSRF protection
4. Review SQL query construction (parameterized vs string concat)
5. Check Dockerfile security (running as root?, exposed ports?)
6. Review migration files for destructive operations
7. Check for any .env, credentials, or secrets committed to git
Report structured findings with file paths and line numbers.`,
    schema: {
      type: 'object',
      properties: {
        subsystem: { type: 'string' },
        files_analyzed: { type: 'number' },
        total_lines: { type: 'number' },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              category: { type: 'string', enum: ['bug', 'security', 'performance', 'code_smell', 'missing_feature'] },
              severity: { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
              title: { type: 'string' },
              description: { type: 'string' },
              file: { type: 'string' },
              line: { type: 'number' },
              recommendation: { type: 'string' }
            },
            required: ['category', 'severity', 'title', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['subsystem', 'findings', 'summary']
    }
  },
]

const explorationResults = await parallel(
  EXPLORATION_TARGETS.map(t => () =>
    agent(t.prompt, { label: t.label, phase: 'Explore', schema: t.schema, model: 'sonnet' })
  )
)

// === Phase 2: Run tests and measure coverage ===
phase('Test')

const testResults = await parallel([
  () => agent(
    `Run the Go test suite for the Baxi project at /home/zzz/project/baxi. Execute:
1. \`cd /home/zzz/project/baxi && go test ./... -count=1 -short 2>&1\` — capture output
2. \`cd /home/zzz/project/baxi && go test -coverprofile=/tmp/baxi_coverage.out ./... -short 2>&1\` — capture coverage
3. \`cd /home/zzz/project/baxi && go tool cover -func=/tmp/baxi_coverage.out 2>&1 | tail -30\` — get coverage summary
4. Identify any failing tests and categorize: pass/fail/skip counts
5. Report which packages have lowest coverage

Report: total tests, pass/fail/skip counts, overall coverage %, per-package coverage for low-coverage packages, and any test failures with error messages.`,
    { label: 'Go Tests', phase: 'Test', model: 'sonnet',
      schema: {
        type: 'object',
        properties: {
          total_tests: { type: 'number' },
          passed: { type: 'number' },
          failed: { type: 'number' },
          skipped: { type: 'number' },
          overall_coverage_pct: { type: 'number' },
          low_coverage_packages: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                package: { type: 'string' },
                coverage_pct: { type: 'number' }
              }
            }
          },
          failures: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                test: { type: 'string' },
                package: { type: 'string' },
                error: { type: 'string' }
              }
            }
          },
          summary: { type: 'string' }
        },
        required: ['total_tests', 'passed', 'failed', 'overall_coverage_pct', 'summary']
      }
    }
  ),
  () => agent(
    `Run the frontend test suite for the Baxi project at /home/zzz/project/baxi/frontend. Execute:
1. \`cd /home/zzz/project/baxi/frontend && npm test -- --run 2>&1\` — capture output
2. Count pass/fail/skip
3. Identify any failing tests

Report: total tests, pass/fail/skip counts, any failures with error messages.`,
    { label: 'Frontend Tests', phase: 'Test', model: 'haiku',
      schema: {
        type: 'object',
        properties: {
          total_tests: { type: 'number' },
          passed: { type: 'number' },
          failed: { type: 'number' },
          skipped: { type: 'number' },
          failures: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                test: { type: 'string' },
                error: { type: 'string' }
              }
            }
          },
          summary: { type: 'string' }
        },
        required: ['total_tests', 'passed', 'failed', 'summary']
      }
    }
  ),
  () => agent(
    `Run the Go build and lint checks for the Baxi project at /home/zzz/project/baxi. Execute:
1. \`cd /home/zzz/project/baxi && go vet ./... 2>&1\` — check for issues
2. \`cd /home/zzz/project/baxi && go build ./... 2>&1\` — check build succeeds
3. Count total Go source files (*.go) and total lines of code (non-test)
4. Count total test files (*_test.go) and test lines
5. Report code-to-test ratio

Report: build status, vet issues, LOC stats (source vs test), test ratio.`,
    { label: 'Build & Lint', phase: 'Test', model: 'haiku',
      schema: {
        type: 'object',
        properties: {
          build_ok: { type: 'boolean' },
          vet_issues: { type: 'number' },
          vet_details: { type: 'string' },
          source_files: { type: 'number' },
          source_lines: { type: 'number' },
          test_files: { type: 'number' },
          test_lines: { type: 'number' },
          test_ratio: { type: 'string' },
          summary: { type: 'string' }
        },
        required: ['build_ok', 'source_files', 'source_lines', 'test_files', 'test_lines', 'summary']
      }
    }
  )
])

// === Phase 3: Synthesize analysis ===
phase('Analyze')

const allFindings = (explorationResults || []).filter(Boolean).flatMap(r => r.findings || [])
const criticalCount = allFindings.filter(f => f.severity === 'critical').length
const highCount = allFindings.filter(f => f.severity === 'high').length
const securityFindings = allFindings.filter(f => f.category === 'security')
const bugFindings = allFindings.filter(f => f.category === 'bug')
const perfFindings = allFindings.filter(f => f.category === 'performance')
const smellFindings = allFindings.filter(f => f.category === 'code_smell')

log(`Exploration complete: ${allFindings.length} findings (${criticalCount} critical, ${highCount} high)`)
log(`Categories: ${securityFindings.length} security, ${bugFindings.length} bugs, ${perfFindings.length} perf, ${smellFindings.length} code smells`)
log(`Test results: Go ${testResults?.[0]?.passed || 0}/${testResults?.[0]?.total_tests || 0} passed, Frontend ${testResults?.[1]?.passed || 0}/${testResults?.[1]?.total_tests || 0} passed`)

const synthesis = await agent(
  `You are synthesizing a comprehensive audit report for the Baxi e-commerce governance platform.

## Exploration Results (from ${explorationResults?.filter(Boolean).length || 0} subsystem analyses):

${(explorationResults || []).filter(Boolean).map(r => `### ${r.subsystem}\n${r.summary}\nKey findings: ${(r.findings || []).filter(f => f.severity === 'critical' || f.severity === 'high').map(f => `\n- [${f.severity}/${f.category}] ${f.title}: ${f.description}`).join('')}`).join('\n\n')}

## Test Results:
- Go: ${testResults?.[0]?.total_tests || 0} tests, ${testResults?.[0]?.passed || 0} passed, ${testResults?.[0]?.failed || 0} failed, ${testResults?.[0]?.overall_coverage_pct || 0}% coverage
- Frontend: ${testResults?.[1]?.total_tests || 0} tests, ${testResults?.[1]?.passed || 0} passed
- Build: ${testResults?.[2]?.build_ok ? 'OK' : 'FAILED'}, Source LOC: ${testResults?.[2]?.source_lines || 0}, Test LOC: ${testResults?.[2]?.test_lines || 0}

## All Findings Summary:
- Critical: ${criticalCount}, High: ${highCount}
- Security: ${securityFindings.length}, Bugs: ${bugFindings.length}, Performance: ${perfFindings.length}, Code Smells: ${smellFindings.length}

Please produce a structured report with these sections:

1. **Project Overview** — what Baxi is, its architecture, current maturity level
2. **Current Status** — build health, test coverage, code quality metrics
3. **Critical & High Findings** — prioritized list of the most important issues
4. **Subsystem Analysis** — brief assessment of each subsystem
5. **Optimization Plan** — concrete, prioritized recommendations organized by:
   - P0 (critical, fix now): security issues, data loss risks, build failures
   - P1 (high, fix soon): bugs, missing error handling, test gaps
   - P2 (medium, plan for next sprint): performance, code smells, missing features
   - P3 (low, backlog): nice-to-haves, minor improvements
6. **Architecture Recommendations** — structural improvements for scalability and maintainability

Be specific with file paths and actionable recommendations. Write in Chinese.`,
  { label: 'Synthesis', phase: 'Analyze', model: 'sonnet' }
)

return {
  exploration: explorationResults?.filter(Boolean).map(r => ({ subsystem: r.subsystem, summary: r.summary, findingCount: (r.findings || []).length })),
  tests: {
    go: testResults?.[0],
    frontend: testResults?.[1],
    build: testResults?.[2]
  },
  totalFindings: allFindings.length,
  criticalFindings: criticalCount,
  highFindings: highCount,
  report: synthesis
}
