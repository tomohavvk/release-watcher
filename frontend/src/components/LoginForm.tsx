import { useState } from 'react'
import { api } from '../hooks/useApi'
import { User } from '../types'

interface Props {
  onLogin: (user: User) => void
}

export default function LoginForm({ onLogin }: Props) {
  const [username, setUsername] = useState('')
  const [code, setCode] = useState('')
  const [step, setStep] = useState<'username' | 'code'>('username')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleRequestCode = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim()) return
    setLoading(true)
    setError('')
    try {
      await api.requestCode(username.trim())
      setStep('code')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send code')
    } finally {
      setLoading(false)
    }
  }

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!code.trim()) return
    setLoading(true)
    setError('')
    try {
      const userData = await api.verifyCode(username.trim(), code.trim())
      onLogin({ id: userData.id, name: userData.name, created_at: userData.created_at })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-8 w-full max-w-md shadow-xl">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-white mb-2">Release Watcher</h1>
          <p className="text-gray-400">Track GitHub releases like a feed</p>
        </div>

        {step === 'username' ? (
          <form onSubmit={handleRequestCode} className="space-y-4">
            <div>
              <label htmlFor="username" className="block text-sm font-medium text-gray-300 mb-1">
                Telegram username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="@username"
                className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                autoFocus
              />
            </div>
            {error && <p className="text-red-400 text-sm">{error}</p>}
            <button
              type="submit"
              disabled={!username.trim() || loading}
              className="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white font-medium rounded-lg transition-colors"
            >
              {loading ? 'Sending...' : 'Send Code'}
            </button>
            <p className="text-gray-500 text-xs text-center">
              First, start the bot <a href="https://t.me/ReleaseWatcherBot" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:underline">@ReleaseWatcherBot</a>
            </p>
          </form>
        ) : (
          <form onSubmit={handleVerify} className="space-y-4">
            <div>
              <label htmlFor="code" className="block text-sm font-medium text-gray-300 mb-1">
                Enter the code from Telegram
              </label>
              <input
                id="code"
                type="text"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="123456"
                className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-center text-2xl tracking-widest"
                autoFocus
                maxLength={6}
              />
            </div>
            {error && <p className="text-red-400 text-sm">{error}</p>}
            <button
              type="submit"
              disabled={!code.trim() || loading}
              className="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white font-medium rounded-lg transition-colors"
            >
              {loading ? 'Verifying...' : 'Log In'}
            </button>
            <button
              type="button"
              onClick={() => { setStep('username'); setCode(''); setError('') }}
              className="w-full py-2 text-gray-400 hover:text-white text-sm transition-colors"
            >
              Back
            </button>
          </form>
        )}
      </div>
    </div>
  )
}
