import { useState, useEffect, useCallback } from 'react'
import { User } from './types'
import { api } from './hooks/useApi'
import LoginForm from './components/LoginForm'
import Sidebar from './components/Sidebar'
import Feed from './components/Feed'

export default function App() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshKey, setRefreshKey] = useState(0)

  useEffect(() => {
    const stored = localStorage.getItem('rw_user')
    if (stored) {
      try {
        const prev = JSON.parse(stored) as User
        setUser(prev)
        api.auth(prev.name).then((fresh) => {
          const u: User = { id: fresh.id, name: fresh.name, created_at: fresh.created_at }
          setUser(u)
          localStorage.setItem('rw_user', JSON.stringify(u))
        }).catch(() => {})
      } catch {
        localStorage.removeItem('rw_user')
      }
    }
    setLoading(false)
  }, [])

  const handleLogin = async (name: string) => {
    try {
      const userData = await api.auth(name)
      const u: User = { id: userData.id, name: userData.name, created_at: userData.created_at }
      setUser(u)
      localStorage.setItem('rw_user', JSON.stringify(u))
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Login failed')
    }
  }

  const handleLogout = () => {
    setUser(null)
    localStorage.removeItem('rw_user')
  }

  const refreshFeed = useCallback(() => {
    setRefreshKey((k) => k + 1)
  }, [])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">Loading...</div>
      </div>
    )
  }

  if (!user) {
    return <LoginForm onLogin={handleLogin} />
  }

  return (
    <div className="flex h-screen">
      <Sidebar
        userId={user.id}
        userName={user.name}
        onLogout={handleLogout}
        onRefreshFeed={refreshFeed}
      />
      <main className="flex-1 overflow-y-auto p-6">
        <div className="max-w-2xl mx-auto">
          <div className="flex items-center justify-between mb-6">
            <h1 className="text-xl font-bold text-white">Release Feed</h1>
            <button
              onClick={refreshFeed}
              className="text-sm text-gray-400 hover:text-white transition-colors"
            >
              Refresh
            </button>
          </div>
          <Feed userId={user.id} refreshKey={refreshKey} />
        </div>
      </main>
    </div>
  )
}
