import { ApiClientError } from "../api/client"
import type { ApiError } from "../api/client"

/**
 * Extracts structured API error details from an unknown error value.
 *
 * React Query sets `error` to the thrown value (ApiClientError in our case).
 * This hook safely extracts the structured fields for display.
 */
export function useApiError(err: unknown): {
  title: string
  message: string
  request_id?: string
  diagnosis?: string
  suggested_action?: string
} {
  if (err instanceof ApiClientError) {
    const ae: ApiError = err.apiError
    // Specific token error → give clear guidance
    if (err.status === 401) {
      return {
        title: "Token 授权失败",
        message: ae.message || "API Token 无效或未配置",
        request_id: ae.request_id,
        diagnosis:
          ae.diagnosis ||
          "API Token 不匹配。请检查左下角 Token 输入框中的值是否与后端 API_BEARER_TOKEN 一致。",
        suggested_action:
          ae.suggested_action ||
          "在左侧栏底部输入正确的 Token 后按回车，或在后端重新设置 API_BEARER_TOKEN 环境变量",
      }
    }
    if (err.status === 500) {
      return {
        title: "服务器内部错误",
        message: ae.message || "API 服务异常",
        request_id: ae.request_id,
        diagnosis: ae.diagnosis || "后端服务处理请求时发生错误",
        suggested_action: ae.suggested_action || "稍后重试，若持续失败请联系技术支持",
      }
    }
    return {
      title: `请求失败 (${err.status})`,
      message: ae.message || "未知错误",
      request_id: ae.request_id,
      diagnosis: ae.diagnosis,
      suggested_action: ae.suggested_action,
    }
  }

  if (err instanceof Error) {
    // Network errors (fetch failed, timeout, etc.)
    if (err.message?.includes("fetch") || err.message?.includes("NetworkError") || err.message?.includes("Failed to")) {
      return {
        title: "网络连接失败",
        message: err.message,
        diagnosis: "无法连接到 API 服务，请确认后端 (localhost:8080) 已启动",
        suggested_action: "运行 docker compose up -d 启动后端服务",
      }
    }
    return {
      title: "请求异常",
      message: err.message || "未知错误",
    }
  }

  return {
    title: "请求失败",
    message: "连接异常",
  }
}
