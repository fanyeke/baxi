import { useQuery } from "@tanstack/react-query"
import { apiClient } from "./client"

export interface CatalogAsset {
  asset_id: string
  asset_type: string
  name: string
  location: string
  description?: string
  grain?: string
  status: string
}

export interface CatalogListResponse {
  items: CatalogAsset[]
  total: number
}

export interface Classification {
  asset_ref: string
  level: string
  rationale: string
  applies_to_fields?: Record<string, string>
}

export interface ClassificationListResponse {
  items: Classification[]
  total: number
}

export interface MarkingInfo {
  mandatory_control: boolean
  access_type: string
  conjunctive: boolean
  inheritance: string[]
  applies_to: string[]
  policy: string
}

export interface MarkingListResponse {
  items: MarkingInfo[]
  total: number
}

export interface LineageNode {
  id: string
  type: string
  label: string
  status: string
}

export interface LineageEdge {
  from: string
  to: string
  transform: string
  transform_type: string
}

export interface LineageResponse {
  nodes: LineageNode[]
  edges: LineageEdge[]
}

export interface CheckpointRule {
  scope: string
  endpoint?: string
  requires_justification: boolean
  prompt?: string
  checkpoint_types?: string[]
}

export interface CheckpointListResponse {
  items: CheckpointRule[]
  total: number
}

export interface HealthCheck {
  id: string
  resource?: string
  description: string
  check_type: string
  severity: string
  validation?: string
}

export interface MonitoringView {
  id: string
  scope: string
  check_type: string
  rule: string
  severity: string
}

export interface HealthResponse {
  monitoring_views: MonitoringView[]
  health_checks: HealthCheck[]
}

export function useCatalog() {
  return useQuery<CatalogListResponse>({
    queryKey: ["governance", "catalog"],
    queryFn: () => apiClient.get<CatalogListResponse>("/governance/catalog"),
  })
}

export function useClassification() {
  return useQuery<ClassificationListResponse>({
    queryKey: ["governance", "class"],
    queryFn: () => apiClient.get<ClassificationListResponse>("/governance/classification"),
  })
}

export function useMarkings() {
  return useQuery<MarkingListResponse>({
    queryKey: ["governance", "markings"],
    queryFn: () => apiClient.get<MarkingListResponse>("/governance/markings"),
  })
}

export function useLineage() {
  return useQuery<LineageResponse>({
    queryKey: ["governance", "lineage"],
    queryFn: () => apiClient.get<LineageResponse>("/governance/lineage"),
  })
}

export function useCheckpoints() {
  return useQuery<CheckpointListResponse>({
    queryKey: ["governance", "chkpts"],
    queryFn: () => apiClient.get<CheckpointListResponse>("/governance/checkpoints"),
  })
}

export function useHealth() {
  return useQuery<HealthResponse>({
    queryKey: ["governance", "health"],
    queryFn: () => apiClient.get<HealthResponse>("/governance/health"),
  })
}
