# FRONTEND KNOWLEDGE BASE

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW
React 19 SPA, Vite 6, TanStack Query 5, Tailwind CSS v4, Radix UI.

## STRUCTURE
```
frontend/
├── src/pages/        # 13 route pages
├── src/components/   # 5 shared UI elements (Layout, ErrorPanel...)
├── src/api/          # apiClient, typed endpoints, types
├── src/hooks/        # empty
├── src/lib/          # empty
└── src/__tests__/    # Vitest setup
```
## WHERE TO LOOK
| Concern | Location | Notes |
|---------|----------|-------|
| Routes | `src/pages/` | 13 pages, one per route |
| Agent Logs | `src/pages/AgentLogs/` | Agent execution logs viewer, MCP activity feed |
| API calls | `src/api/` | `client.ts`, `governance.ts`, `types.ts` |
| Shared UI | `src/components/` | Barrel export via `index.ts` |
| Layout | `src/components/Layout.tsx` | Nav shell, wraps all routes |

## PATTERNS & CONVENTIONS
- **`@/` path alias** in tsconfig + vitest config
- **Colocated `__tests__/`**: each page has `PageName.test.tsx`
- **Styling**: CVA + `tailwind-merge` (`cn()`), `clsx` for conditionals
- **Test helper**: `renderWithQueryClient(ui)` — wraps in QueryClientProvider, retries off
- **`verbatimModuleSyntax`**: `import type` for type-only imports
- Radix UI primitives (Dialog, DropdownMenu, Select, Tabs, Toast, Tooltip)
- `lucide-react` for all icons
- No ESLint, no Prettier — TypeScript compiler only

## ANTI-PATTERNS
- `hooks/` and `lib/` exist but are empty
- No coverage config in vitest.config.ts
- `renderWithQueryClient` duplicated inline across every test file
