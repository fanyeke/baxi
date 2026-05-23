interface EmptyStateProps {
  title: string
  description?: string
  icon?: string
}

export function EmptyState({ title, description, icon = "○" }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <span className="text-3xl text-muted-foreground/40 mb-3">{icon}</span>
      <p className="text-muted-foreground font-medium">{title}</p>
      {description && <p className="text-xs text-muted-foreground mt-1">{description}</p>}
    </div>
  )
}
