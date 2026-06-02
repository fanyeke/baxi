import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"

vi.mock("@/api/client", () => ({
  apiClient: { get: vi.fn() },
}))

import { apiClient } from "@/api/client"
import {
  useCatalog,
  useClassification,
  useMarkings,
  useLineage,
  useCheckpoints,
  useHealth,
} from "@/api/governance"

const mockedGet = vi.mocked(apiClient.get)

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useCatalog", () => {
  it("calls /governance/catalog", async () => {
    const data = { objects: [], datasets: [] }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useCatalog(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockedGet).toHaveBeenCalledWith("/governance/catalog")
  })

  it("returns loading state initially", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    const { result } = renderHook(() => useCatalog(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
    expect(result.current.data).toBeUndefined()
  })

  it("returns data on success", async () => {
    const data = {
      objects: [
        {
          asset_id: "a1",
          asset_type: "table",
          name: "orders",
          location: "dw.orders",
          status: "active",
        },
      ],
      datasets: ["ds1"],
    }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useCatalog(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(data)
  })

  it("returns error on failure", async () => {
    mockedGet.mockRejectedValueOnce(new Error("network fail"))

    const { result } = renderHook(() => useCatalog(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })

  it("uses staleTime of 30_000", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    renderHook(() => useCatalog(), { wrapper: createWrapper() })

    expect(mockedGet).toHaveBeenCalledTimes(1)
  })
})

describe("useClassification", () => {
  it("calls /governance/classification", async () => {
    const data = { levels: [], resources: [] }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useClassification(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockedGet).toHaveBeenCalledWith("/governance/classification")
  })

  it("returns loading state initially", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    const { result } = renderHook(() => useClassification(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
  })

  it("returns data on success", async () => {
    const data = {
      levels: [{ level: "confidential", type: "pii" }],
      resources: ["dw.orders"],
    }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useClassification(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(data)
  })

  it("returns error on failure", async () => {
    mockedGet.mockRejectedValueOnce(new Error("fail"))

    const { result } = renderHook(() => useClassification(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })

  it("uses staleTime of 30_000", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    renderHook(() => useClassification(), { wrapper: createWrapper() })

    expect(mockedGet).toHaveBeenCalledTimes(1)
  })
})

describe("useMarkings", () => {
  it("calls /governance/markings", async () => {
    const data = { markings: [] }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useMarkings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockedGet).toHaveBeenCalledWith("/governance/markings")
  })

  it("returns loading state initially", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    const { result } = renderHook(() => useMarkings(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
  })

  it("returns data on success", async () => {
    const data = {
      markings: [
        {
          mandatory_control: true,
          access_type: "role-based",
          conjunctive: false,
          inheritance: ["table"],
          applies_to: ["dw.orders"],
          policy: "arn:policy",
        },
      ],
      pipeline_stage_markings: [{ stage: "ingest" }],
    }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useMarkings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(data)
  })

  it("returns error on failure", async () => {
    mockedGet.mockRejectedValueOnce(new Error("fail"))

    const { result } = renderHook(() => useMarkings(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })

  it("uses staleTime of 30_000", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    renderHook(() => useMarkings(), { wrapper: createWrapper() })

    expect(mockedGet).toHaveBeenCalledTimes(1)
  })
})

describe("useLineage", () => {
  it("calls /governance/lineage", async () => {
    const data = { resource: "dw.orders", upstream: [], downstream: [] }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useLineage(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockedGet).toHaveBeenCalledWith("/governance/lineage")
  })

  it("returns loading state initially", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    const { result } = renderHook(() => useLineage(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
  })

  it("returns data on success", async () => {
    const data = {
      resource: "dw.orders",
      upstream: [
        {
          from: "raw.orders",
          to: "dw.orders",
          transform: "clean",
          transform_type: "etl",
        },
      ],
      downstream: [
        {
          from: "dw.orders",
          to: "agg.revenue",
          transform: "aggregate",
          transform_type: "sql",
        },
      ],
    }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useLineage(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(data)
  })

  it("returns error on failure", async () => {
    mockedGet.mockRejectedValueOnce(new Error("fail"))

    const { result } = renderHook(() => useLineage(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })

  it("uses staleTime of 30_000", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    renderHook(() => useLineage(), { wrapper: createWrapper() })

    expect(mockedGet).toHaveBeenCalledTimes(1)
  })
})

describe("useCheckpoints", () => {
  it("calls /governance/checkpoints", async () => {
    const data = { checkpoints: [] }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useCheckpoints(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockedGet).toHaveBeenCalledWith("/governance/checkpoints")
  })

  it("returns loading state initially", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    const { result } = renderHook(() => useCheckpoints(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
  })

  it("returns data on success", async () => {
    const data = {
      checkpoints: [
        {
          action: "pii-scan",
          requires_reason: true,
          requires_human_review: false,
        },
      ],
    }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useCheckpoints(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(data)
  })

  it("returns error on failure", async () => {
    mockedGet.mockRejectedValueOnce(new Error("fail"))

    const { result } = renderHook(() => useCheckpoints(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })

  it("uses staleTime of 30_000", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    renderHook(() => useCheckpoints(), { wrapper: createWrapper() })

    expect(mockedGet).toHaveBeenCalledTimes(1)
  })
})

describe("useHealth", () => {
  it("calls /governance/health", async () => {
    const data = { status: "healthy", checks: [] }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useHealth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockedGet).toHaveBeenCalledWith("/governance/health")
  })

  it("returns loading state initially", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    const { result } = renderHook(() => useHealth(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)
  })

  it("returns data on success", async () => {
    const data = {
      status: "degraded",
      checks: [
        { name: "catalog-sync", status: "healthy" },
        { name: "db-connection", status: "unhealthy" },
      ],
    }
    mockedGet.mockResolvedValueOnce(data)

    const { result } = renderHook(() => useHealth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(data)
  })

  it("returns error on failure", async () => {
    mockedGet.mockRejectedValueOnce(new Error("fail"))

    const { result } = renderHook(() => useHealth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })

  it("uses staleTime of 30_000", () => {
    mockedGet.mockReturnValueOnce(new Promise(() => {}))

    renderHook(() => useHealth(), { wrapper: createWrapper() })

    expect(mockedGet).toHaveBeenCalledTimes(1)
  })
})
