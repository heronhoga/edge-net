export interface User {
  id: string
  name: string
  email: string
  provider: 'google' | 'admin'
}

export interface SessionInfo {
  user: User | null
  csrfToken: string
  googleEnabled: boolean
  adminEnabled: boolean
}

export function initials(user: User) {
  return user.name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
    .toUpperCase()
}
