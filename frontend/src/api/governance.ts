import { useQuery } from "@tanstack/react-query"
import { apiClient } from "./client"

export interface CatalogObject {
  object_type: string
  source_dataset: string
  primary_key: string
  properties_count: number
  links_count: number
}

export interface CatalogDataset {
  dataset: string
  schema: string
  table: string
}

export interface CatalogResponse {
  objects: CatalogObject[]
  datasets: CatalogDataset[]
}

export interface Classification {
  asset_ref: string
  level: string
  rationale: string
  applies_to_fields?: Record<string, string>
}

export interface ClassificationResponse {
  classifications: Classification[]
}

export interface MarkingInfo {
  mandatory_control: boolean
  access_type: string
  conjunctive: boolean
  inheritance: string[]
  applies_to: string[]
  policy: string
}

export interface MarkingResponse {
  markings: Record<string, MarkingInfo>
  pipeline_stage_markings: Array<Record<string, unknown>>
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

export interface CheckpointResponse {
  checkpoints: Record<string, CheckpointRule>
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
  return useQuery<CatalogResponse>({
    queryKey: ["governance", "catalog"],
    queryFn: () => apiClient.get<CatalogResponse>("/governance/catalog"),
    staleTime: 30_000,
  })
}

export function useClassification() {
  return useQuery<ClassificationResponse>({
    queryKey: ["governance", "class"],
    queryFn: () => apiClient.get<ClassificationResponse>("/governance/classification"),
    staleTime: 30_000,
  })
}

export function useMarkings() {
  return useQuery<MarkingResponse>({
    queryKey: ["governance", "markings"],
    queryFn: () => apiClient.get<MarkingResponse>("/governance/markings"),
    staleTime: 30_000,
  })
}

export function useLineage() {
  return useQuery<LineageResponse>({
    queryKey: ["governance", "lineage"],
    queryFn: () => apiClient.get<LineageResponse>("/governance/lineage"),
    staleTime: 30_000,
  })
}

export function useCheckpoints() {
  return useQuery<CheckpointResponse>({
    queryKey: ["governance", "chkpts"],
    queryFn: () => apiClient.get<CheckpointResponse>("/governance/checkpoints"),
    staleTime: 30_000,
  })
}

export function useHealth() {
  return useQuery<HealthResponse>({
    queryKey: ["governance", "health"],
    queryFn: () => apiClient.get<HealthResponse>("/governance/health"),
    staleTime: 30_000,
  })
}
