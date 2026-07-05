import { useState, useEffect, useCallback } from 'react'
import { Organization, Repository } from '../types'
import { api } from '../hooks/useApi'

interface Props {
  userId: number
  userName: string
  onLogout: () => void
  onRefreshFeed: () => void
}

export default function Sidebar({ userId, userName, onLogout, onRefreshFeed }: Props) {
  const [orgs, setOrgs] = useState<Organization[]>([])
  const [orgInput, setOrgInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [expandedOrg, setExpandedOrg] = useState<number | null>(null)
  const [repos, setRepos] = useState<Repository[]>([])
  const [reposLoading, setReposLoading] = useState(false)

  const loadOrgs = useCallback(async () => {
    try {
      const data = await api.listFollowing(userId)
      setOrgs(data)
    } catch {
      // ignore
    }
  }, [userId])

  useEffect(() => {
    loadOrgs()
  }, [loadOrgs])

  const handleFollow = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orgInput.trim()) return
    setLoading(true)
    try {
      await api.follow(userId, orgInput.trim())
      setOrgInput('')
      await loadOrgs()
      onRefreshFeed()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to follow')
    } finally {
      setLoading(false)
    }
  }

  const handleUnfollow = async (orgId: number) => {
    try {
      await api.unfollow(userId, orgId)
      await loadOrgs()
      onRefreshFeed()
    } catch {
      // ignore
    }
  }

  const toggleOrgRepos = async (orgId: number) => {
    if (expandedOrg === orgId) {
      setExpandedOrg(null)
      setRepos([])
      return
    }
    setExpandedOrg(orgId)
    setReposLoading(true)
    try {
      const data = await api.listRepos(orgId, userId)
      setRepos(data)
    } catch {
      setRepos([])
    } finally {
      setReposLoading(false)
    }
  }

  const handleToggleMute = async (repo: Repository) => {
    try {
      if (repo.muted) {
        await api.unmute(userId, repo.id)
      } else {
        await api.mute(userId, repo.id)
      }
      const data = await api.listRepos(expandedOrg!, userId)
      setRepos(data)
      onRefreshFeed()
    } catch {
      // ignore
    }
  }

  return (
    <aside className="w-80 bg-gray-900 border-r border-gray-800 h-screen flex flex-col">
      <div className="p-4 border-b border-gray-800">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-lg font-bold text-white">Release Watcher</h2>
            <p className="text-sm text-gray-400">{userName}</p>
          </div>
          <button
            onClick={onLogout}
            className="text-xs text-gray-500 hover:text-gray-300 transition-colors"
          >
            logout
          </button>
        </div>

        <form onSubmit={handleFollow} className="flex gap-2">
          <input
            type="text"
            value={orgInput}
            onChange={(e) => setOrgInput(e.target.value)}
            placeholder="org or user name"
            className="flex-1 px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="submit"
            disabled={!orgInput.trim() || loading}
            className="px-3 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 text-white text-sm rounded-lg transition-colors"
          >
            {loading ? '...' : 'Follow'}
          </button>
        </form>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-1">
        <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
          Following ({orgs.length})
        </h3>

        {orgs.map((org) => (
          <div key={org.id}>
            <div className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-gray-800 group">
              <button
                onClick={() => toggleOrgRepos(org.id)}
                className="flex items-center gap-2 text-sm text-gray-300 hover:text-white flex-1 text-left"
              >
                <span className="text-gray-600">{expandedOrg === org.id ? '▼' : '▶'}</span>
                {org.name}
              </button>
              <button
                onClick={() => handleUnfollow(org.id)}
                className="text-gray-600 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-all text-xs"
              >
                unfollow
              </button>
            </div>

            {expandedOrg === org.id && (
              <div className="ml-6 space-y-0.5 mb-2">
                {reposLoading ? (
                  <p className="text-xs text-gray-500 py-1">Loading repos...</p>
                ) : repos.length === 0 ? (
                  <p className="text-xs text-gray-500 py-1">
                    No repos yet. Waiting for first poll...
                  </p>
                ) : (
                  repos.map((repo) => (
                    <div
                      key={repo.id}
                      className="flex items-center justify-between py-1.5 px-2 rounded hover:bg-gray-800"
                    >
                      <span
                        className={`text-xs truncate ${
                          repo.muted ? 'text-gray-600 line-through' : 'text-gray-400'
                        }`}
                      >
                        {repo.name}
                      </span>
                      <button
                        onClick={() => handleToggleMute(repo)}
                        className={`text-xs flex-shrink-0 ml-2 ${
                          repo.muted
                            ? 'text-yellow-600 hover:text-yellow-400'
                            : 'text-gray-600 hover:text-yellow-400'
                        }`}
                      >
                        {repo.muted ? 'unmute' : 'mute'}
                      </button>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        ))}

        {orgs.length === 0 && (
          <p className="text-sm text-gray-500 text-center py-8">
            Follow an organization or user to start watching releases
          </p>
        )}
      </div>
    </aside>
  )
}
