"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { setup, setToken, checkSetupStatus } from "@/lib/auth"
import {
  Shield,
  Lock,
  KeyRound,
  ChevronRight,
  Check,
  ShieldCheck,
  List,
  Loader2,
  ArrowLeft,
} from "lucide-react"

const BLOCKLIST_OPTIONS = [
  {
    id: "stevenblack",
    name: "StevenBlack Unified",
    url: "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
    format: "hosts",
    description: "Ads + malware — most popular, 100K+ domains",
    defaultChecked: true,
  },
  {
    id: "oisd-small",
    name: "OISD Small",
    url: "https://small.oisd.nl/domainswild",
    format: "domains",
    description: "Balanced blocking — minimal false positives",
    defaultChecked: false,
  },
  {
    id: "hagezi-light",
    name: "Hagezi Light",
    url: "https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/hosts/light.txt",
    format: "hosts",
    description: "Lightweight — fast, low resource usage",
    defaultChecked: false,
  },
]

const STEPS = [
  { label: "Set Password", icon: Lock },
  { label: "Blocklists", icon: List },
]

function PasswordStrength({ password }: { password: string }) {
  let strength = 0
  if (password.length >= 8) strength++
  if (password.length >= 12) strength++
  if (/[A-Z]/.test(password) && /[a-z]/.test(password)) strength++
  if (/[0-9]/.test(password) && /[^a-zA-Z0-9]/.test(password)) strength++

  const labels = ["Weak", "Fair", "Good", "Strong"]
  const colors = ["bg-red-500", "bg-amber-500", "bg-hydra-teal/60", "bg-hydra-teal"]

  if (password.length === 0) return null

  return (
    <div className="space-y-2 pt-1">
      <div className="flex justify-between items-center">
        <span className="text-[10px] font-semibold text-slate-500 uppercase tracking-wider font-headline">
          Security Strength
        </span>
        <span className={`text-[10px] font-semibold uppercase tracking-wider font-headline ${strength >= 3 ? "text-hydra-teal" : strength >= 2 ? "text-amber-500" : "text-red-400"}`}>
          {labels[Math.min(strength, 3)]}
        </span>
      </div>
      <div className="grid grid-cols-4 gap-1.5 h-1.5">
        {[0, 1, 2, 3].map((i) => (
          <div
            key={i}
            className={`rounded-full transition-colors ${i <= strength - 1 ? colors[Math.min(strength - 1, 3)] : "bg-hydra-border"}`}
          />
        ))}
      </div>
    </div>
  )
}

export default function SetupPage() {
  const router = useRouter()
  const [step, setStep] = useState(1)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState("")

  // Step 1: Password
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")

  // Step 2: Blocklists
  const [selectedBlocklists, setSelectedBlocklists] = useState<string[]>(
    BLOCKLIST_OPTIONS.filter((b) => b.defaultChecked).map((b) => b.id)
  )

  useEffect(() => {
    checkSetupStatus().then((status) => {
      if (status === "complete") {
        router.replace("/login")
      } else if (status === "unreachable") {
        setError("Cannot reach HydraDNS API — is the server running?")
        setLoading(false)
      } else {
        setLoading(false)
      }
    })
  }, [router])

  function toggleBlocklist(id: string) {
    setSelectedBlocklists((prev) =>
      prev.includes(id) ? prev.filter((b) => b !== id) : [...prev, id]
    )
  }

  function handleNext() {
    setError("")
    if (step === 1) {
      if (password.length < 8) {
        setError("Password must be at least 8 characters")
        return
      }
      if (password !== confirmPassword) {
        setError("Passwords do not match")
        return
      }
    }
    setStep(step + 1)
  }

  async function handleSubmit() {
    setError("")
    setSubmitting(true)

    const blocklists = BLOCKLIST_OPTIONS.filter((b) =>
      selectedBlocklists.includes(b.id)
    ).map((b) => ({ id: b.id, name: b.name, url: b.url, format: b.format }))

    try {
      const token = await setup({ password, blocklists })
      setToken(token)
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Setup failed")
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-6 w-6 animate-spin text-hydra-teal" />
          <p className="text-sm text-muted-foreground font-mono">Connecting...</p>
        </div>
      </div>
    )
  }

  return (
    <div
      className="flex min-h-screen flex-col items-center justify-center bg-background p-6"
      style={{
        backgroundImage:
          "radial-gradient(circle at 2px 2px, rgba(255,255,255,0.03) 1px, transparent 0)",
        backgroundSize: "40px 40px",
      }}
    >
      {/* Top bar branding */}
      <header className="fixed top-0 left-0 right-0 flex items-center justify-between px-6 py-4 bg-background/80 backdrop-blur-sm z-10">
        <div className="text-xl font-black text-hydra-teal tracking-tight font-headline">
          HydraDNS
        </div>
        <Shield className="h-5 w-5 text-slate-500" />
      </header>

      <main className="w-full max-w-md">
        {/* Setup wizard card */}
        <div className="bg-card border border-border rounded-xl shadow-2xl p-8 md:p-10">
          {/* Logo + Step indicator */}
          <div className="flex flex-col items-center mb-8">
            <div className="w-16 h-16 bg-hydra-teal/10 rounded-full flex items-center justify-center mb-6">
              <ShieldCheck className="h-8 w-8 text-hydra-teal" />
            </div>

            {/* Step dots */}
            <div className="flex gap-3 items-center mb-2">
              {STEPS.map((_, i) => (
                <div
                  key={i}
                  className={`h-1.5 w-8 rounded-full transition-colors ${
                    i + 1 <= step ? "bg-hydra-teal" : "bg-hydra-border"
                  }`}
                />
              ))}
            </div>
            <span className="text-[10px] font-headline font-bold uppercase tracking-[0.2em] text-hydra-teal">
              Setup Wizard
            </span>
          </div>

          {/* Step heading */}
          <div className="text-center mb-8">
            <h1 className="text-2xl font-headline font-bold text-white mb-2">
              {step === 1 && "Secure Your Gateway"}
              {step === 2 && "Choose Blocklists"}
            </h1>
            <p className="text-slate-400 text-sm leading-relaxed">
              {step === 1 &&
                "Create an admin password to protect your HydraDNS dashboard"}
              {step === 2 &&
                "Select threat intelligence feeds to block ads, trackers, and malware"}
            </p>
          </div>

          {/* Step 1: Password */}
          {step === 1 && (
            <div className="space-y-5">
              <div className="space-y-2">
                <label className="block text-xs font-headline font-semibold text-slate-400 uppercase tracking-wider">
                  New Password
                </label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Minimum 8 characters"
                    autoFocus
                    className="w-full bg-background border border-border rounded-lg pl-10 pr-4 py-3 text-white placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-hydra-teal focus:border-hydra-teal transition-all font-mono text-sm"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="block text-xs font-headline font-semibold text-slate-400 uppercase tracking-wider">
                  Confirm Password
                </label>
                <div className="relative">
                  <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
                  <input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="Re-enter password"
                    className="w-full bg-background border border-border rounded-lg pl-10 pr-4 py-3 text-white placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-hydra-teal focus:border-hydra-teal transition-all font-mono text-sm"
                  />
                </div>
              </div>

              <PasswordStrength password={password} />
            </div>
          )}

          {/* Step 2: Blocklists */}
          {step === 2 && (
            <div className="space-y-3">
              {BLOCKLIST_OPTIONS.map((bl) => {
                const isSelected = selectedBlocklists.includes(bl.id)
                return (
                  <button
                    key={bl.id}
                    type="button"
                    onClick={() => toggleBlocklist(bl.id)}
                    className={`w-full flex items-start gap-3 rounded-lg border p-4 text-left transition-all ${
                      isSelected
                        ? "border-hydra-teal/50 bg-hydra-teal/5"
                        : "border-border bg-background hover:border-border/80"
                    }`}
                  >
                    <div
                      className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border transition-colors ${
                        isSelected
                          ? "bg-hydra-teal border-hydra-teal"
                          : "border-slate-600 bg-transparent"
                      }`}
                    >
                      {isSelected && (
                        <Check className="h-3 w-3 text-[#1A1D23]" />
                      )}
                    </div>
                    <div className="space-y-0.5">
                      <span className="text-sm font-medium text-white">
                        {bl.name}
                      </span>
                      <p className="text-xs text-slate-500">{bl.description}</p>
                    </div>
                  </button>
                )
              })}
            </div>
          )}

          {/* Error message */}
          {error && (
            <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3">
              <p className="text-sm text-red-400">{error}</p>
            </div>
          )}

          {/* Actions */}
          <div className="mt-8 flex gap-3">
            {step > 1 && (
              <button
                type="button"
                onClick={() => { setError(""); setStep(step - 1) }}
                className="flex items-center gap-1 rounded-lg border border-border px-4 py-3 text-sm font-headline font-semibold text-slate-400 transition-colors hover:text-white hover:border-slate-500"
              >
                <ArrowLeft className="h-4 w-4" />
                Back
              </button>
            )}
            <div className="flex-1" />
            {step < 2 ? (
              <button
                type="button"
                onClick={handleNext}
                className="flex items-center justify-center gap-2 rounded-lg bg-hydra-teal px-6 py-3 text-sm font-headline font-bold text-[#1A1D23] shadow-lg shadow-hydra-teal/20 transition-all hover:bg-hydra-teal/90 active:scale-[0.98]"
              >
                Continue
                <ChevronRight className="h-4 w-4" />
              </button>
            ) : (
              <button
                type="button"
                onClick={handleSubmit}
                disabled={submitting}
                className="flex items-center justify-center gap-2 rounded-lg bg-hydra-teal px-6 py-3 text-sm font-headline font-bold text-[#1A1D23] shadow-lg shadow-hydra-teal/20 transition-all hover:bg-hydra-teal/90 active:scale-[0.98] disabled:opacity-60 disabled:cursor-not-allowed"
              >
                {submitting ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Setting up...
                  </>
                ) : (
                  <>
                    Complete Setup
                    <ShieldCheck className="h-4 w-4" />
                  </>
                )}
              </button>
            )}
          </div>

          {/* Card footer */}
          <div className="mt-8 pt-6 border-t border-border/50 text-center">
            <span className="text-xs font-mono text-slate-500 uppercase tracking-tight">
              Step {step} of {STEPS.length}
            </span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="mt-10 text-center text-xs text-slate-600">
        HydraDNS Security
      </footer>
    </div>
  )
}
