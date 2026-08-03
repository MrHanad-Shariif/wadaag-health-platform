import { apiClient } from '../../api/client'
import type { Branch, BranchInput, CreateFacilityInput, Department, DepartmentInput, Facility, UpdateFacilityInput } from './types'

export function listFacilities() {
  return apiClient.get<Facility[]>('/api/v1/facilities')
}

export function getFacility(facilityId: string) {
  return apiClient.get<Facility>(`/api/v1/facilities/${facilityId}`)
}

export function createFacility(input: CreateFacilityInput) {
  return apiClient.post<Facility>('/api/v1/facilities', input)
}

export function updateFacility(facilityId: string, input: UpdateFacilityInput) {
  return apiClient.patch<Facility>(`/api/v1/facilities/${facilityId}`, input)
}

export function listBranches(facilityId: string) {
  return apiClient.get<Branch[]>(`/api/v1/facilities/${facilityId}/branches`)
}

export function createBranch(facilityId: string, input: BranchInput) {
  return apiClient.post<Branch>(`/api/v1/facilities/${facilityId}/branches`, input)
}

// updateBranch/deleteBranch aren't nested under facilityID — the backend
// mounts them at /facilities/branches/{branchID} (see Handler.Routes in
// backend/internal/facilities/handler.go).
export function updateBranch(branchId: string, input: BranchInput) {
  return apiClient.patch<Branch>(`/api/v1/facilities/branches/${branchId}`, input)
}

export function deleteBranch(branchId: string) {
  return apiClient.del<void>(`/api/v1/facilities/branches/${branchId}`)
}

export function listDepartments(facilityId: string) {
  return apiClient.get<Department[]>(`/api/v1/facilities/${facilityId}/departments`)
}

export function createDepartment(facilityId: string, input: DepartmentInput) {
  return apiClient.post<Department>(`/api/v1/facilities/${facilityId}/departments`, input)
}

// Same non-nested mount as branches — /facilities/departments/{departmentID}.
export function updateDepartment(departmentId: string, input: DepartmentInput) {
  return apiClient.patch<Department>(`/api/v1/facilities/departments/${departmentId}`, input)
}

export function deleteDepartment(departmentId: string) {
  return apiClient.del<void>(`/api/v1/facilities/departments/${departmentId}`)
}
