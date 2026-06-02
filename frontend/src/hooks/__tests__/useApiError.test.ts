import { describe, it, expect } from "vitest"
import { renderHook } from "@testing-library/react"
import { ApiClientError } from "@/api/client"
import { useApiError } from "@/hooks/useApiError"

function makeApiClientError(
  status: number,
  overrides: Partial<{
    request_id: string
    error_code: string
    message: string
    diagnosis: string
    suggested_action: string
  }> = {},
) {
  return new ApiClientError(status, {
    request_id: "req-abc",
    error_code: "ERR",
    message: "default message",
    diagnosis: "default diagnosis",
    suggested_action: "default action",
    ...overrides,
  })
}

describe("ApiClientError 401", () => {
  it('returns title "Token 授权失败"', () => {
    const err = makeApiClientError(401, {
      message: "Token expired",
      diagnosis: "Token mismatch",
      suggested_action: "Enter a valid token",
    })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("Token 授权失败")
    expect(result.current.message).toBe("Token expired")
    expect(result.current.request_id).toBe("req-abc")
    expect(result.current.diagnosis).toBe("Token mismatch")
    expect(result.current.suggested_action).toBe("Enter a valid token")
  })

  it("falls back to default message when API message is empty", () => {
    const err = makeApiClientError(401, { message: "" })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.message).toBe("API Token 无效或未配置")
  })

  it("falls back to default diagnosis and suggested_action when empty", () => {
    const err = makeApiClientError(401, {
      diagnosis: "",
      suggested_action: "",
    })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.diagnosis).toContain("API Token 不匹配")
    expect(result.current.suggested_action).toContain("在左侧栏底部输入正确的 Token")
  })
})

describe("ApiClientError 500", () => {
  it('returns title "服务器内部错误"', () => {
    const err = makeApiClientError(500, {
      message: "Internal failure",
      diagnosis: "DB crashed",
      suggested_action: "Retry later",
    })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("服务器内部错误")
    expect(result.current.message).toBe("Internal failure")
    expect(result.current.request_id).toBe("req-abc")
    expect(result.current.diagnosis).toBe("DB crashed")
    expect(result.current.suggested_action).toBe("Retry later")
  })

  it("falls back to defaults when API fields are empty", () => {
    const err = makeApiClientError(500, {
      message: "",
      diagnosis: "",
      suggested_action: "",
    })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.message).toBe("API 服务异常")
    expect(result.current.diagnosis).toBe("后端服务处理请求时发生错误")
    expect(result.current.suggested_action).toBe("稍后重试，若持续失败请联系技术支持")
  })
})

describe("Other ApiClientError", () => {
  it("returns title with status code for 403", () => {
    const err = makeApiClientError(403, { message: "Forbidden" })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("请求失败 (403)")
    expect(result.current.message).toBe("Forbidden")
    expect(result.current.request_id).toBe("req-abc")
  })

  it("returns title with status code for 404", () => {
    const err = makeApiClientError(404, {
      message: "Not found",
      diagnosis: "Resource missing",
      suggested_action: "Check URL",
    })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("请求失败 (404)")
    expect(result.current.message).toBe("Not found")
    expect(result.current.diagnosis).toBe("Resource missing")
    expect(result.current.suggested_action).toBe("Check URL")
  })

  it('falls back to "未知错误" when message is empty', () => {
    const err = makeApiClientError(422, { message: "" })

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.message).toBe("未知错误")
  })
})

describe("Network errors", () => {
  it('returns title "网络连接失败" for fetch error', () => {
    const err = new Error("fetch failed")

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("网络连接失败")
    expect(result.current.message).toBe("fetch failed")
    expect(result.current.diagnosis).toContain("localhost:8080")
    expect(result.current.suggested_action).toContain("docker compose")
  })

  it('returns title "网络连接失败" for NetworkError', () => {
    const err = new Error("NetworkError when attempting to fetch resource")

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("网络连接失败")
  })

  it('returns title "网络连接失败" for "Failed to" prefix', () => {
    const err = new Error("Failed to fetch")

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("网络连接失败")
  })
})

describe("Generic Error", () => {
  it('returns title "请求异常"', () => {
    const err = new Error("something went wrong")

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("请求异常")
    expect(result.current.message).toBe("something went wrong")
  })

  it('falls back to "未知错误" when message is empty', () => {
    const err = new Error("")

    const { result } = renderHook(() => useApiError(err))

    expect(result.current.title).toBe("请求异常")
    expect(result.current.message).toBe("未知错误")
  })
})

describe("Unknown error", () => {
  it('returns title "请求失败" for string error', () => {
    const { result } = renderHook(() => useApiError("oops"))

    expect(result.current.title).toBe("请求失败")
    expect(result.current.message).toBe("连接异常")
  })

  it('returns title "请求失败" for null', () => {
    const { result } = renderHook(() => useApiError(null))

    expect(result.current.title).toBe("请求失败")
    expect(result.current.message).toBe("连接异常")
  })

  it('returns title "请求失败" for undefined', () => {
    const { result } = renderHook(() => useApiError(undefined))

    expect(result.current.title).toBe("请求失败")
    expect(result.current.message).toBe("连接异常")
  })

  it('returns title "请求失败" for numeric error', () => {
    const { result } = renderHook(() => useApiError(42))

    expect(result.current.title).toBe("请求失败")
    expect(result.current.message).toBe("连接异常")
  })
})
