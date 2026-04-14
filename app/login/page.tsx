"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { Shield, Lock, ArrowRight, Loader2, AlertTriangle } from "lucide-react"
import { login, setToken, checkSetupStatus, getToken } from "@/lib/auth"

export default function LoginPage() {
  const router = useRouter()
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [apiReachable, setApiReachable] = useState(true)

  useEffect(() => {
    // If already logged in, go to dashboard
    if (getToken()) {
      router.replace("/dashboard")
      return
    }
    // Check if setup is needed
    checkSetupStatus().then((status) => {
      if (status === "needs_setup") {
        router.replace("/setup")
      } else if (status === "unreachable") {
        setApiReachable(false)
        setError("Cannot reach HydraDNS API — is the server running?")
        setLoading(false)
      } else {
        setLoading(false)
      }
    })
  }, [router])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError("")
    setSubmitting(true)
    try {
      const token = await login(password)
      setToken(token)
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed")
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#1A1D23]">
        <Loader2 className="w-6 h-6 text-[#00D4AA] animate-spin" />
      </div>
    )
  }

  return (
    <div
      className="min-h-screen flex flex-col items-center justify-center p-6 text-slate-200"
      style={{
        backgroundColor: "#1A1D23",
        backgroundImage: [
          "radial-gradient(at 0% 0%, rgba(0, 212, 170, 0.05) 0px, transparent 50%)",
          "radial-gradient(at 100% 0%, rgba(14, 165, 233, 0.03) 0px, transparent 50%)",
          "linear-gradient(rgba(0, 212, 170, 0.02) 1px, transparent 1px)",
          "linear-gradient(90deg, rgba(0, 212, 170, 0.02) 1px, transparent 1px)",
        ].join(", "),
        backgroundSize: "100% 100%, 100% 100%, 40px 40px, 40px 40px",
      }}
    >
      <main className="flex-grow flex items-center justify-center w-full max-w-md">
        <div className="w-full">
          {/* Login Card */}
          <div className="bg-[#22262E] border border-[#2D333D] rounded-xl p-8 shadow-2xl relative overflow-hidden">
            {/* Subtle Top Accent Line */}
            <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-[#00D4AA]/50 via-[#00D4AA] to-[#00D4AA]/50 opacity-50" />

            {/* Card Header */}
            <div className="flex flex-col items-center text-center mb-10">
              <div className="w-16 h-16 bg-[#1A1D23] border border-[#00D4AA]/20 rounded-xl flex items-center justify-center mb-4 shadow-inner">
                <Shield className="w-8 h-8 text-[#00D4AA]" fill="currentColor" />
              </div>
              <h1 className="font-headline text-3xl font-bold tracking-tight text-[#00D4AA] mb-1">
                HydraDNS
              </h1>
              <p className="text-slate-400 font-headline text-sm uppercase tracking-wider">
                DNS Security Gateway
              </p>
            </div>

            {/* Form */}
            <form onSubmit={handleSubmit} className="space-y-6">
              <div>
                <label
                  htmlFor="password"
                  className="block font-headline text-xs font-medium text-slate-400 uppercase tracking-widest mb-2 px-1"
                >
                  Admin Password
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Lock className="w-4 h-4 text-slate-500" />
                  </div>
                  <input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Enter your password"
                    autoFocus
                    required
                    disabled={!apiReachable}
                    className="block w-full bg-[#1A1D23] border border-[#2D333D] text-slate-200 text-sm rounded-lg focus:ring-[#00D4AA] focus:border-[#00D4AA] pl-10 pr-3 py-3.5 transition-all placeholder:text-slate-600 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
                  />
                </div>
              </div>

              {/* Error message */}
              {error && (
                <div className={`flex items-start gap-2 text-sm rounded-lg px-3 py-2.5 ${
                  apiReachable
                    ? "text-red-400 bg-red-400/5 border border-red-400/10"
                    : "text-amber-400 bg-amber-400/5 border border-amber-400/10"
                }`}>
                  <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
                  <span className="font-mono text-xs leading-relaxed">{error}</span>
                </div>
              )}

              <button
                type="submit"
                disabled={submitting || !apiReachable}
                className="w-full bg-[#00D4AA] hover:bg-[#00BF9A] text-[#1A1D23] font-headline font-bold py-4 px-6 rounded-lg transition-all transform active:scale-[0.98] flex items-center justify-center gap-2 group shadow-lg shadow-[#00D4AA]/10 disabled:opacity-50 disabled:cursor-not-allowed disabled:active:scale-100"
              >
                {submitting ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <span>Sign In</span>
                    <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
                  </>
                )}
              </button>
            </form>

            {/* Internal Footer */}
            <div className="mt-10 pt-6 border-t border-[#2D333D]/50 text-center">
              <p className="text-slate-500 text-xs font-medium italic">
                Protect your network at the DNS layer
              </p>
            </div>
          </div>

          {/* Appliance Meta Info */}
          <div className="mt-8 flex justify-between items-center px-4">
            <div className="flex items-center gap-2">
              <div className={`w-2 h-2 rounded-full ${
                apiReachable
                  ? "bg-green-500 animate-pulse"
                  : "bg-amber-500 animate-pulse"
              }`} />
              <span className="text-[10px] font-mono text-slate-500 uppercase tracking-tighter">
                {apiReachable ? "System Online" : "System Unreachable"}
              </span>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
