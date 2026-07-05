import { Release } from '../types'

interface Props {
  release: Release
}

function timeAgo(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diff = now - then
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  const months = Math.floor(days / 30)
  return `${months}mo ago`
}

export default function ReleaseCard({ release }: Props) {
  const isRelease = release.type === 'release'

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-5 hover:border-gray-700 transition-colors">
      <div className="flex items-start gap-3">
        {release.author_avatar ? (
          <img
            src={release.author_avatar}
            alt={release.author}
            className="w-10 h-10 rounded-full flex-shrink-0"
          />
        ) : (
          <div className="w-10 h-10 rounded-full bg-gray-700 flex items-center justify-center flex-shrink-0">
            <span className="text-gray-400 text-sm">
              {isRelease ? 'R' : 'T'}
            </span>
          </div>
        )}

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <a
              href={`https://github.com/${release.repo_name}`}
              target="_blank"
              rel="noopener noreferrer"
              className="font-semibold text-white hover:text-blue-300 transition-colors"
            >
              {release.repo_name}
            </a>
            <span
              className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                isRelease
                  ? 'bg-green-900/50 text-green-400 border border-green-800'
                  : 'bg-blue-900/50 text-blue-400 border border-blue-800'
              }`}
            >
              {isRelease ? 'release' : 'tag'}
            </span>
            <span className="text-gray-500 text-sm">{timeAgo(release.published_at)}</span>
          </div>

          <a
            href={release.html_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-400 hover:text-blue-300 font-medium mt-1 inline-block"
          >
            {release.tag_name}
            {release.name && release.name !== release.tag_name && (
              <span className="text-gray-400 font-normal"> - {release.name}</span>
            )}
          </a>

          {release.body && (
            <p className="text-gray-400 text-sm mt-2 line-clamp-3 whitespace-pre-wrap">
              {release.body.length > 300
                ? release.body.slice(0, 300) + '...'
                : release.body}
            </p>
          )}

          <div className="flex items-center gap-3 mt-3 text-xs text-gray-500">
            {release.author && <span>by {release.author}</span>}
            <span>{release.org_name}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
