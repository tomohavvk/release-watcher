import { ApiResponse } from '../types'

const BASE = '/api/v1'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) throw new Error(body.error)
  return body.data
}

export const api = {
  requestCode: (username: string) =>
    request<{ status: string }>('/auth/request', {
      method: 'POST',
      body: JSON.stringify({ username }),
    }),

  verifyCode: (username: string, code: string) =>
    request<{ id: number; name: string; created_at: string }>('/auth/verify', {
      method: 'POST',
      body: JSON.stringify({ username, code }),
    }),

  getUser: (name: string) =>
    request<{ id: number; name: string; created_at: string }>(`/users/${name}`),

  getFeed: (userId: number, limit = 50, offset = 0) =>
    request<Array<import('../types').Release>>(
      `/feed?user_id=${userId}&limit=${limit}&offset=${offset}`,
    ),

  follow: (userId: number, orgName: string) =>
    request<import('../types').Organization>('/following', {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, org_name: orgName }),
    }),

  unfollow: (userId: number, orgId: number) =>
    request<{ status: string }>(`/following/${orgId}?user_id=${userId}`, {
      method: 'DELETE',
    }),

  listFollowing: (userId: number) =>
    request<Array<import('../types').Organization>>(`/following?user_id=${userId}`),

  listRepos: (orgId: number, userId: number) =>
    request<Array<import('../types').Repository>>(
      `/repos?org_id=${orgId}&user_id=${userId}`,
    ),

  mute: (userId: number, repoId: number) =>
    request<{ status: string }>('/mutes', {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, repo_id: repoId }),
    }),

  unmute: (userId: number, repoId: number) =>
    request<{ status: string }>(`/mutes/${repoId}?user_id=${userId}`, {
      method: 'DELETE',
    }),

  listMutes: (userId: number) =>
    request<Array<import('../types').MutedRepo>>(`/mutes?user_id=${userId}`),
}
