import { useState, useEffect, useCallback } from 'react'
import { Release } from '../types'
import { api } from '../hooks/useApi'
import ReleaseCard from './ReleaseCard'

interface Props {
  userId: number
  refreshKey: number
}

export default function Feed({ userId, refreshKey }: Props) {
  const [releases, setReleases] = useState<Release[]>([])
  const [loading, setLoading] = useState(true)
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)

  const loadFeed = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.getFeed(userId, 50, 0)
      setReleases(data)
      setOffset(50)
      setHasMore(data.length === 50)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [userId])

  useEffect(() => {
    loadFeed()
  }, [loadFeed, refreshKey])

  const loadMore = async () => {
    try {
      const data = await api.getFeed(userId, 50, offset)
      setReleases((prev) => [...prev, ...data])
      setOffset((prev) => prev + 50)
      setHasMore(data.length === 50)
    } catch {
      // ignore
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">Loading feed...</div>
      </div>
    )
  }

  if (releases.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-center">
        <p className="text-gray-400 text-lg mb-2">No releases yet</p>
        <p className="text-gray-500 text-sm">
          Follow organizations in the sidebar to start watching their releases.
          <br />
          New releases will appear here after the next poll cycle.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {releases.map((release) => (
        <ReleaseCard key={release.id} release={release} />
      ))}
      {hasMore && (
        <button
          onClick={loadMore}
          className="w-full py-3 text-sm text-gray-400 hover:text-white bg-gray-900 border border-gray-800 rounded-xl hover:border-gray-700 transition-colors"
        >
          Load more
        </button>
      )}
    </div>
  )
}
