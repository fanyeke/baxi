import { Component } from "react"
import type { ErrorInfo, ReactNode } from "react"

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error("[ErrorBoundary]", error, errorInfo)
  }

  handleReload = (): void => {
    window.location.reload()
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div className="flex items-center justify-center min-h-screen p-4">
          <div className="w-full max-w-md p-4 border border-destructive/50 bg-destructive/10 rounded-lg">
            <p className="font-semibold text-destructive">
              应用发生了错误
            </p>
            <p className="text-sm text-muted-foreground mt-1">
              {this.state.error?.message || "未知错误，请尝试刷新页面"}
            </p>
            <button
              onClick={this.handleReload}
              className="mt-3 px-3 py-1 border rounded text-xs hover:bg-muted"
            >
              重新加载
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
