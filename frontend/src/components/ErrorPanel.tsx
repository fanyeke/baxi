interface ErrorPanelProps {
  title: string
  message: string
  request_id?: string
  diagnosis?: string
  suggested_action?: string
  onRetry?: () => void
  variant?: "inline" | "full"
}

export function ErrorPanel({
  title, message, request_id, diagnosis, suggested_action, onRetry, variant = "full",
}: ErrorPanelProps) {
  return (
    <div className={`p-4 border border-destructive/50 bg-destructive/10 rounded-lg ${variant === "full" ? "" : "text-sm"}`}>
      <p className="font-semibold text-destructive">{title}</p>
      <p className={`text-muted-foreground mt-1 ${variant === "full" ? "text-sm" : "text-xs"}`}>{message}</p>
      {diagnosis && (
        <p className="text-xs text-muted-foreground mt-1">
          <span className="font-medium">诊断:</span> {diagnosis}
        </p>
      )}
      {suggested_action && (
        <p className="text-xs text-muted-foreground mt-1">
          <span className="font-medium">建议:</span> {suggested_action}
        </p>
      )}
      {request_id && (
        <p className="text-xs text-muted-foreground mt-1 font-mono">
          Request ID: {request_id.slice(0, 16)}...
        </p>
      )}
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-2 px-3 py-1 border rounded text-xs hover:bg-muted"
        >
          重试
        </button>
      )}
    </div>
  )
}
