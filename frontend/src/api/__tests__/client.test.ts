import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"

// Mock global fetch before importing client
const mockFetch = vi.fn()
vi.stubGlobal("fetch", mockFetch)

import { ApiClientError } from "../client"

// We need to test getToken and request indirectly since they're not exported.
// We'll test through the exported apiClient methods.

describe("getToken()", () => {
  beforeEach(() => {
    sessionStorage.clear()
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.stubGlobal("fetch", mockFetch)
  })

  it("reads token from sessionStorage", async () => {
    // Import dynamically to get fresh module access
    const { apiClient } = await import("../client")

    sessionStorage.setItem("API_BEARER_TOKEN", "test-token-123")
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: "ok" }),
    })

    await apiClient.get("/test")

    expect(mockFetch).toHaveBeenCalledWith("/api/v1/test", {
      headers: {
        "Content-Type": "application/json",
        "Authorization": "Bearer test-token-123",
      },
      signal: expect.any(AbortSignal),
    })
  })
})

describe("request() adds Bearer Authorization header", () => {
  beforeEach(() => {
    sessionStorage.clear()
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.stubGlobal("fetch", mockFetch)
  })

  it("adds Bearer token from sessionStorage when present", async () => {
    const { apiClient } = await import("../client")

    sessionStorage.setItem("API_BEARER_TOKEN", "my-secret-token")
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: "ok" }),
    })

    await apiClient.get("/health")

    const callArgs = mockFetch.mock.calls[0]
    expect(callArgs[0]).toBe("/api/v1/health")
    expect(callArgs[1].headers["Authorization"]).toBe("Bearer my-secret-token")
  })

  it("omits Authorization header when no token", async () => {
    const { apiClient } = await import("../client")

    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: "ok" }),
    })

    await apiClient.get("/health")

    const callArgs = mockFetch.mock.calls[0]
    expect(callArgs[1].headers).not.toHaveProperty("Authorization")
  })
})

describe("ApiClientError", () => {
  it("captures error_code, diagnosis, suggested_action from API response", () => {
    const apiError = {
      request_id: "req-123",
      error_code: "INVALID_TOKEN",
      message: "Token expired",
      diagnosis: "The provided token has expired",
      suggested_action: "Refresh your token",
    }

    const err = new ApiClientError(401, apiError)

    expect(err).toBeInstanceOf(Error)
    expect(err.status).toBe(401)
    expect(err.apiError.error_code).toBe("INVALID_TOKEN")
    expect(err.apiError.diagnosis).toBe("The provided token has expired")
    expect(err.apiError.suggested_action).toBe("Refresh your token")
    expect(err.message).toBe("Token expired")
  })
})
