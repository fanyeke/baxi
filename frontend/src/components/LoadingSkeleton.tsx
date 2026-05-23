interface LoadingSkeletonProps {
  type: "table" | "cards" | "text"
  count?: number
}

export function LoadingSkeleton({ type, count }: LoadingSkeletonProps) {
  const n = count ?? (type === "cards" ? 4 : type === "table" ? 3 : 3)

  if (type === "cards") {
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {Array.from({ length: n }).map((_, i) => (
          <div key={i} className="h-24 bg-muted animate-pulse rounded-lg" />
        ))}
      </div>
    )
  }

  if (type === "table") {
    return (
      <div className="space-y-2">
        {Array.from({ length: n }).map((_, i) => (
          <div key={i} className="h-10 bg-muted animate-pulse rounded" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {Array.from({ length: n }).map((_, i) => (
        <div key={i} className="h-4 bg-muted animate-pulse rounded" style={{ width: `${80 - i * 15}%` }} />
      ))}
    </div>
  )
}
