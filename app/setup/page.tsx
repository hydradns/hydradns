"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { setup, setToken, checkSetupStatus } from "@/lib/auth"

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
        <p className="text-muted-foreground">Loading...</p>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-lg">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">HydraDNS Setup</CardTitle>
          <CardDescription>
            Step {step} of 2 —{" "}
            {step === 1 && "Set admin password"}
            {step === 2 && "Choose blocklists"}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {step === 1 && (
            <>
              <div className="space-y-2">
                <Label htmlFor="password">Admin Password</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Minimum 8 characters"
                  autoFocus
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm">Confirm Password</Label>
                <Input
                  id="confirm"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Re-enter password"
                />
              </div>
            </>
          )}

          {step === 2 && (
            <div className="space-y-3">
              <Label>Blocklist Sources</Label>
              {BLOCKLIST_OPTIONS.map((bl) => (
                <div
                  key={bl.id}
                  className="flex items-start space-x-3 rounded-lg border p-3"
                >
                  <Checkbox
                    id={bl.id}
                    checked={selectedBlocklists.includes(bl.id)}
                    onCheckedChange={() => toggleBlocklist(bl.id)}
                  />
                  <div className="space-y-0.5">
                    <label
                      htmlFor={bl.id}
                      className="text-sm font-medium cursor-pointer"
                    >
                      {bl.name}
                    </label>
                    <p className="text-xs text-muted-foreground">
                      {bl.description}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}

          {error && <p className="text-sm text-destructive">{error}</p>}

          <div className="flex gap-2 pt-2">
            {step > 1 && (
              <Button variant="outline" onClick={() => setStep(step - 1)}>
                Back
              </Button>
            )}
            <div className="flex-1" />
            {step < 2 ? (
              <Button onClick={handleNext}>Next</Button>
            ) : (
              <Button onClick={handleSubmit} disabled={submitting}>
                {submitting ? "Setting up..." : "Complete Setup"}
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
