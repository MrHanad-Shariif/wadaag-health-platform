export interface Profile {
  user_id: string
  photo_url: string | null
  bio: string | null
  languages: string[]
  availability_status: string | null
  is_online: boolean
  last_seen_at: string | null
}

export interface UpdateProfileInput {
  bio?: string
  languages?: string[]
  availability_status?: string
}
