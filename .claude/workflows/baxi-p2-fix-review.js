export const meta = {
  name: 'baxi-p2-fix-review',
  description: 'Fix remaining P2 bugs and iterate until code review passes with no medium/high issues',
  phases: [
    { title: 'Fix', detail: 'Fix all remaining bugs' },
    { title: 'Verify', detail: 'Build and test' },
    { title: 'Review', detail: 'Code review for remaining issues' },
  ],
}

// === Phase 1: Fix all remaining bugs ===
phase('Fix')

const fixResults = await parallel([
  // Agent 1: High severity - Worker race condition (Opus)
  () => agent(
    [
      'Fix the double-execution race condition in the Baxi DispatchWorker at /home/zzz/project/baxi.',
      '',
      'File: internal/worker/dispatch_worker.go',
      '',
      '## Problem',
      'The Run() method uses a ticker that fires every PollInterval (30s). processBatch() is synchronous.',
      'If a batch takes longer than PollInterval, the next tick fires and calls processBatch concurrently.',
      'There is NO synchronization. Two goroutines will fetch overlapping events and dispatch them twice.',
      '',
      '## Fix',
      'Add a sync.Mutex or channel-based guard. The simplest correct approach:',
      '',
      '1. Add to the DispatchWorker struct: mu sync.Mutex',
      '2. In the Run() select, wrap processBatch:',
      '',
      '```go',
      'case <-ticker.C:',
      '    if w.mu.TryLock() {',
      '        w.processBatch(ctx)',
      '        w.mu.Unlock()',
      '    }',
      '```',
      '',
      'Alternatively, use a channel-based approach:',
      '```go',
      'var processing chan struct{} = make(chan struct{}, 1)',
      'case <-ticker.C:',
      '    select {',
      '    case processing <- struct{}{}:',
      '        go func() {',
      '            w.processBatch(ctx)',
      '            <-processing',
      '        }()',
      '    default:',
      '        // skip this tick, previous batch still running',
      '    }',
      '```',
      '',
      'Choose the approach that best fits the existing code style.',
      'Make sure to add "sync" to the import block.',
      '',
      '## Verification',
      'Run: cd /home/zzz/project/baxi && go build ./...',
      'Run: cd /home/zzz/project/baxi && go test ./internal/worker/... -count=1 -v -race',
    ].join('\n'),
    { label: 'Worker Race Fix', model: 'sonnet' }
  ),

  // Agent 2: Medium bugs - roundTo + N+1 + ContextBuilder flag + dedup
  () => agent(
    [
      'Fix 4 medium/low bugs in the Baxi project at /home/zzz/project/baxi.',
      '',
      '## Bug A: roundTo negative number rounding',
      'File: internal/alert/engine.go, lines 323-329',
      '',
      'Current implementation uses int64(v*pow+0.5) which breaks for negative numbers.',
      'Replace the function body with math.Round:',
      '```go',
      'func roundTo(v float64, decimals int) float64 {',
      '    pow := 1.0',
      '    for i := 0; i < decimals; i++ {',
      '        pow *= 10',
      '    }',
      '    return math.Round(v*pow) / pow',
      '}',
      '```',
      'Add "math" to imports if not present.',
      '',
      '## Bug B: N+1 getLatestDate query',
      'File: internal/alert/engine.go, around line 40-56',
      '',
      'getLatestDate() is called inside the rule evaluation loop. It always returns the same result.',
      'Move the call to BEFORE the loop:',
      '',
      'Find the EvaluateGlobalRules function. Before the for loop over rules, add:',
      '```go',
      'lastDate, err := e.getLatestDate(ctx, tx)',
      'if err != nil {',
      '    return nil, fmt.Errorf("get latest date: %w", err)',
      '}',
      '```',
      'Then remove the getLatestDate call from inside the loop and use the pre-fetched lastDate.',
      '',
      '## Bug C: Wire up V2 ContextBuilder feature flag',
      'File: internal/api/handler_factories.go, around line 178',
      '',
      'The feature flag FlagNewContextBuilder is defined in internal/feature/flags.go but never checked.',
      'V2 ContextBuilder is never instantiated in production.',
      '',
      'In handler_factories.go, find where V1 ContextBuilder is created (around line 178).',
      'Add a conditional that checks the feature flag:',
      '',
      '1. Import "baxi/internal/feature" if not already imported',
      '2. Check: if feature.IsEnabled(feature.FlagNewContextBuilder) {',
      '    // instantiate V2 (need to check V2 constructor signature)',
      '    // ctxBuilder = decision.NewContextBuilderV2(...)',
      '  } else {',
      '    // keep V1',
      '    ctxBuilder = decision.NewContextBuilder(...)',
      '  }',
      '',
      'Read context_builder_v2.go to understand V2 constructor parameters.',
      'If V2 needs dependencies not available in the current scope, add them to the Server struct.',
      '',
      '## Bug D: Deduplicate LLMSafeContext construction',
      'File: internal/service/decision_service.go, lines 122-151',
      '',
      'The DecisionContext -> LLMSafeContext mapping is duplicated between here and engine.go.',
      'Export the mapping function from the decision package:',
      '',
      '1. In internal/decision/engine.go, find buildLLMSafeContext and rename to BuildLLMSafeContext (export it)',
      '2. Update the call site in engine.go itself (line 87)',
      '3. In decision_service.go, replace the inline mapping with: llmSafeCtx := decision.BuildLLMSafeContext(decCtx)',
      '',
      'Read both files first to understand the exact function signatures.',
      '',
      '## Verification',
      'Run: cd /home/zzz/project/baxi && go build ./...',
      'Run: cd /home/zzz/project/baxi && go test ./internal/alert/... ./internal/decision/... ./internal/service/... ./internal/api/... -count=1 -v',
    ].join('\n'),
    { label: 'Medium Bugs Fix', model: 'sonnet' }
  ),

  // Agent 3: Frontend cleanup
  () => agent(
    [
      'Remove unused npm dependencies from the Baxi frontend at /home/zzz/project/baxi/frontend.',
      '',
      '## Packages to Remove',
      'These 8 packages have zero imports in frontend/src/:',
      '- @radix-ui/react-toast',
      '- @radix-ui/react-tooltip',
      '- @radix-ui/react-select',
      '- @radix-ui/react-dropdown-menu',
      '- class-variance-authority',
      '- clsx',
      '- lucide-react',
      '- tailwind-merge',
      '',
      '## How to Remove',
      'Run from /home/zzz/project/baxi/frontend:',
      '```bash',
      'npm uninstall @radix-ui/react-toast @radix-ui/react-tooltip @radix-ui/react-select @radix-ui/react-dropdown-menu class-variance-authority clsx lucide-react tailwind-merge',
      '```',
      '',
      '## Verification',
      '1. cd /home/zzz/project/baxi/frontend && npm test -- --run',
      '2. npx tsc --noEmit',
      '3. Confirm the packages are gone from package.json',
    ].join('\n'),
    { label: 'Frontend Cleanup', model: 'haiku' }
  ),
])

log('Phase 1 complete: ' + (fixResults?.filter(Boolean).length || 0) + ' agents finished')

// === Phase 2: Verify ===
phase('Verify')

const verifyResults = await parallel([
  () => agent(
    [
      'Run Go build and test verification for the Baxi project at /home/zzz/project/baxi.',
      '',
      'Execute:',
      '1. cd /home/zzz/project/baxi && go build ./... 2>&1',
      '2. cd /home/zzz/project/baxi && go vet ./... 2>&1',
      '3. cd /home/zzz/project/baxi && go test ./... -count=1 -short 2>&1',
      '',
      'Report: build_ok, vet_ok, tests_passed, tests_failed, failures list.',
    ].join('\n'),
    { label: 'Go Verify', model: 'sonnet',
      schema: {
        type: 'object',
        properties: {
          build_ok: { type: 'boolean' },
          vet_ok: { type: 'boolean' },
          tests_passed: { type: 'number' },
          tests_failed: { type: 'number' },
          failures: { type: 'array', items: { type: 'object', properties: { test: { type: 'string' }, error: { type: 'string' } } } },
          summary: { type: 'string' }
        },
        required: ['build_ok', 'vet_ok', 'tests_passed', 'tests_failed', 'summary']
      }
    }
  ),
  () => agent(
    [
      'Run frontend verification for /home/zzz/project/baxi/frontend.',
      '1. npm test -- --run',
      '2. npx tsc --noEmit',
      'Report: tests_passed, tests_failed, tsc_ok.',
    ].join('\n'),
    { label: 'Frontend Verify', model: 'haiku',
      schema: {
        type: 'object',
        properties: {
          tests_passed: { type: 'number' },
          tests_failed: { type: 'number' },
          tsc_ok: { type: 'boolean' },
          summary: { type: 'string' }
        },
        required: ['tests_passed', 'tests_failed', 'tsc_ok', 'summary']
      }
    }
  )
])

// === Phase 3: Code Review ===
phase('Review')

const reviewResult = await agent(
  [
    'Perform a code review of the Baxi project at /home/zzz/project/baxi looking for remaining medium and high severity bugs.',
    '',
    'Focus on:',
    '1. Security vulnerabilities (SQL injection, auth bypass, XSS, secrets)',
    '2. Concurrency bugs (race conditions, deadlocks)',
    '3. Data loss risks (missing transactions, silent error swallowing)',
    '4. Logic errors that could cause incorrect behavior',
    '',
    'For each finding, classify as: critical, high, medium, or low.',
    'Only report critical, high, and medium findings.',
    'Ignore code smells, style issues, and low-severity items.',
    '',
    'Check these specific areas that were recently fixed:',
    '- internal/api/middleware/identity.go (JWT verification)',
    '- internal/repository/governance/repository.go (SQL injection)',
    '- internal/governance/access_policy.go (deny-overrides)',
    '- internal/worker/dispatch_worker.go (race condition fix)',
    '- internal/alert/engine.go (roundTo + N+1 fix)',
    '- internal/api/handler_factories.go (handler caching)',
    '',
    'Report: list of findings with severity, file, description. If no medium/high findings, say "PASS - no medium/high issues found".',
  ].join('\n'),
  { label: 'Code Review', model: 'sonnet',
    schema: {
      type: 'object',
      properties: {
        status: { type: 'string', enum: ['pass', 'fail'] },
        findings: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              severity: { type: 'string', enum: ['critical', 'high', 'medium'] },
              file: { type: 'string' },
              description: { type: 'string' },
              recommendation: { type: 'string' }
            },
            required: ['severity', 'file', 'description']
          }
        },
        summary: { type: 'string' }
      },
      required: ['status', 'summary']
    }
  }
)

// Generate final report
const goV = verifyResults?.[0] || {}
const feV = verifyResults?.[1] || {}

const report = await agent(
  [
    'Generate a final status report for the Baxi project in Chinese.',
    '',
    '## What was fixed this round (P2):',
    '- Worker double-execution race condition (sync.Mutex guard)',
    '- roundTo negative number rounding (math.Round)',
    '- N+1 getLatestDate query (hoisted before loop)',
    '- V2 ContextBuilder feature flag wired up',
    '- LLMSafeContext construction deduplicated',
    '- 8 unused npm dependencies removed',
    '',
    '## Verification:',
    '- Go: build=' + goV.build_ok + ', vet=' + goV.vet_ok + ', passed=' + goV.tests_passed + ', failed=' + goV.tests_failed,
    '- Frontend: passed=' + feV.tests_passed + ', tsc=' + feV.tsc_ok,
    '',
    '## Code Review Result:',
    '- Status: ' + (reviewResult?.status || 'unknown'),
    '- Findings: ' + (reviewResult?.findings?.length || 0) + ' medium/high issues',
    '- ' + (reviewResult?.summary || ''),
    '',
    'Write a concise report covering:',
    '1. All fixes made across P0/P1/P2 rounds',
    '2. Current project health status',
    '3. Code review verdict',
    '4. Any remaining items (low severity only)',
  ].join('\n'),
  { label: 'Final Report', model: 'sonnet' }
)

return {
  fixes: fixResults?.filter(Boolean).length || 0,
  verification: { go: goV, frontend: feV },
  review: reviewResult,
  report: report
}
