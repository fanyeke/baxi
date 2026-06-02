import { useApiError } from "@/hooks/useApiError"
import { ErrorPanel } from "./ErrorPanel"

export function ApiErrorPanel({ error }: { error: unknown }) {
  const errInfo = useApiError(error)

  if (!error) return null

  return (
    <ErrorPanel
      title={errInfo.title}
      message={errInfo.message}
      request_id={errInfo.request_id}
      diagnosis={errInfo.diagnosis}
      suggested_action={errInfo.suggested_action}
    />
  )
}
