import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"

export function useLogQuery<T>(queryKey: string, endpoint: string, params?: Record<string, string>) {
  return useQuery({
    queryKey: [queryKey],
    queryFn: () => apiClient.get<T>(endpoint, params),
  })
}
