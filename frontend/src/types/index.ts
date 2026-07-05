export interface User {
  id: number
  name: string
  created_at: string
}

export interface Organization {
  id: number
  name: string
  last_polled_at?: string
  created_at: string
}

export interface Repository {
  id: number
  org_id: number
  name: string
  full_name: string
  created_at: string
  muted?: boolean
}

export interface Release {
  id: number
  repo_id: number
  tag_name: string
  name: string
  body: string
  html_url: string
  type: 'release' | 'tag'
  author: string
  author_avatar: string
  published_at: string
  created_at: string
  repo_name: string
  org_name: string
}

export interface MutedRepo {
  user_id: number
  repo_id: number
  repo_name: string
  created_at: string
}

export interface ApiResponse<T> {
  data: T
  error?: string
}
