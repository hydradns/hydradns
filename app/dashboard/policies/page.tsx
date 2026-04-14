"use client"

import { useEffect, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { getPolicies, createPolicy, deletePolicy } from "@/lib/api"
import type { Policy, PolicyListData } from "@/lib/types"
import {
  Plus, Trash2, Shield, ShieldCheck, ArrowRightLeft,
  Globe, ScrollText, ChevronDown,
} from "lucide-react"

const actionConfig: Record<string, { label: string; badge: string; icon: string; bg: string }> = {
  BLOCK: {
    label: "BLOCK",
    badge: "bg-red-500/10 text-red-500 border border-red-500/20",
    icon: "bg-red-500/10 text-red-500 border-red-500/20",
    bg: "red",
  },
  ALLOW: {
    label: "ALLOW",
    badge: "bg-green-500/10 text-green-500 border border-green-500/20",
    icon: "bg-green-500/10 text-green-500 border-green-500/20",
    bg: "green",
  },
  REDIRECT: {
    label: "REDIRECT",
    badge: "bg-blue-500/10 text-blue-500 border border-blue-500/20",
    icon: "bg-blue-500/10 text-blue-500 border-blue-500/20",
    bg: "blue",
  },
}

function ActionIcon({ action }: { action: string }) {
  const cls = "w-5 h-5"
  switch (action) {
    case "BLOCK": return <Shield className={cls} />
    case "ALLOW": return <ShieldCheck className={cls} />
    case "REDIRECT": return <ArrowRightLeft className={cls} />
    default: return <Shield className={cls} />
  }
}

export default function PoliciesPage() {
  const [data, setData] = useState<PolicyListData | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [deleting, setDeleting] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<string | null>(null)

  // Form state
  const [formId, setFormId] = useState("")
  const [formName, setFormName] = useState("")
  const [formDescription, setFormDescription] = useState("")
  const [formAction, setFormAction] = useState("BLOCK")
  const [formDomains, setFormDomains] = useState("")
  const [formCategory, setFormCategory] = useState("")
  const [formPriority, setFormPriority] = useState("100")

  const [error, setError] = useState<string | null>(null)

  const fetchData = () => {
    getPolicies().then((d) => { setData(d); setError(null) }).catch((e) => setError(e.message))
  }

  useEffect(() => { fetchData() }, [])

  const resetForm = () => {
    setFormId("")
    setFormName("")
    setFormDescription("")
    setFormAction("BLOCK")
    setFormDomains("")
    setFormCategory("")
    setFormPriority("100")
    setFormError(null)
  }

  const handleCreate = async () => {
    const domains = formDomains.split(/[,\n]/).map((d) => d.trim()).filter(Boolean)
    if (!formId || !formName || domains.length === 0) {
      setFormError("ID, Name, and at least one domain are required")
      return
    }
    setSubmitting(true)
    setFormError(null)
    try {
      await createPolicy({
        id: formId,
        name: formName,
        description: formDescription || undefined,
        category: formCategory || undefined,
        action: formAction,
        domains,
        priority: parseInt(formPriority) || 100,
      })
      resetForm()
      setShowForm(false)
      fetchData()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : "Failed to create policy")
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    setDeleting(id)
    try {
      await deletePolicy(id)
      fetchData()
    } finally {
      setDeleting(null)
    }
  }

  return (
    <>
      <header className="flex flex-wrap gap-3 min-h-20 py-4 shrink-0 items-center border-b border-border">
        <div className="flex flex-1 items-center gap-2">
          <SidebarTrigger className="-ms-1" />
          <div className="max-lg:hidden lg:contents">
            <Separator orientation="vertical" className="me-2 data-[orientation=vertical]:h-4" />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink href="/dashboard">Home</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>Policies</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <button
          onClick={() => { setShowForm(!showForm); if (showForm) resetForm() }}
          className="flex items-center gap-2 bg-[#00D4AA] text-background font-bold px-4 py-2 rounded-lg text-sm hover:bg-[#00f5c4] transition-colors shadow-lg shadow-[#00D4AA]/20"
        >
          <Plus className="w-4 h-4" />
          {showForm ? "Cancel" : "Add Policy"}
        </button>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        {/* Error state */}
        {error && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-400">
            {error}
          </div>
        )}

        {/* Inline Add Form */}
        {showForm && (
          <section>
            <div className="bg-card rounded-xl border border-border overflow-hidden relative">
              <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-[#00D4AA] to-[#0EA5E9]" />
              <div className="p-6">
                <div className="flex items-center gap-3 mb-6">
                  <ScrollText className="w-5 h-5 text-[#00D4AA]" />
                  <h2 className="font-headline text-xl font-bold text-foreground">Configure New Policy</h2>
                </div>

                {formError && (
                  <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
                    {formError}
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
                  <div className="md:col-span-4">
                    <label className="block text-xs font-bold text-muted-foreground uppercase tracking-widest mb-2">
                      Policy ID
                    </label>
                    <input
                      type="text"
                      className="w-full bg-background border border-border rounded-lg px-4 py-3 text-foreground focus:outline-none focus:border-[#00D4AA] transition-colors placeholder:text-muted-foreground/50 text-sm"
                      placeholder="e.g. block-social"
                      value={formId}
                      onChange={(e) => setFormId(e.target.value)}
                    />
                  </div>
                  <div className="md:col-span-4">
                    <label className="block text-xs font-bold text-muted-foreground uppercase tracking-widest mb-2">
                      Policy Name
                    </label>
                    <input
                      type="text"
                      className="w-full bg-background border border-border rounded-lg px-4 py-3 text-foreground focus:outline-none focus:border-[#00D4AA] transition-colors placeholder:text-muted-foreground/50 text-sm"
                      placeholder="e.g. Block Social Media"
                      value={formName}
                      onChange={(e) => setFormName(e.target.value)}
                    />
                  </div>
                  <div className="md:col-span-4">
                    <label className="block text-xs font-bold text-muted-foreground uppercase tracking-widest mb-2">
                      Action
                    </label>
                    <div className="relative">
                      <select
                        className="w-full appearance-none bg-background border border-border rounded-lg px-4 py-3 text-foreground focus:outline-none focus:border-[#00D4AA] transition-colors text-sm"
                        value={formAction}
                        onChange={(e) => setFormAction(e.target.value)}
                      >
                        <option value="BLOCK">Block</option>
                        <option value="ALLOW">Allow</option>
                        <option value="REDIRECT">Redirect</option>
                      </select>
                      <ChevronDown className="w-4 h-4 absolute right-3 top-3.5 text-muted-foreground pointer-events-none" />
                    </div>
                  </div>
                  <div className="md:col-span-12">
                    <label className="block text-xs font-bold text-muted-foreground uppercase tracking-widest mb-2">
                      Domains (one per line)
                    </label>
                    <textarea
                      className="w-full bg-background border border-border rounded-lg px-4 py-3 text-foreground focus:outline-none focus:border-[#00D4AA] transition-colors placeholder:text-muted-foreground/50 font-mono text-sm leading-relaxed"
                      placeholder={"facebook.com\ntiktok.com\nads.example.net"}
                      rows={4}
                      value={formDomains}
                      onChange={(e) => setFormDomains(e.target.value)}
                    />
                  </div>
                  <div className="md:col-span-3">
                    <label className="block text-xs font-bold text-muted-foreground uppercase tracking-widest mb-2">
                      Priority
                    </label>
                    <input
                      type="number"
                      className="w-full bg-background border border-border rounded-lg px-4 py-3 text-foreground focus:outline-none focus:border-[#00D4AA] transition-colors text-sm"
                      value={formPriority}
                      onChange={(e) => setFormPriority(e.target.value)}
                    />
                    <p className="text-[10px] text-muted-foreground mt-2 italic">Higher numbers override lower priorities.</p>
                  </div>
                  <div className="md:col-span-3">
                    <label className="block text-xs font-bold text-muted-foreground uppercase tracking-widest mb-2">
                      Category
                    </label>
                    <input
                      type="text"
                      className="w-full bg-background border border-border rounded-lg px-4 py-3 text-foreground focus:outline-none focus:border-[#00D4AA] transition-colors placeholder:text-muted-foreground/50 text-sm"
                      placeholder="e.g. social, security"
                      value={formCategory}
                      onChange={(e) => setFormCategory(e.target.value)}
                    />
                  </div>
                  <div className="md:col-span-6">
                    <label className="block text-xs font-bold text-muted-foreground uppercase tracking-widest mb-2">
                      Description
                    </label>
                    <input
                      type="text"
                      className="w-full bg-background border border-border rounded-lg px-4 py-3 text-foreground focus:outline-none focus:border-[#00D4AA] transition-colors placeholder:text-muted-foreground/50 text-sm"
                      placeholder="Optional description"
                      value={formDescription}
                      onChange={(e) => setFormDescription(e.target.value)}
                    />
                  </div>
                  <div className="md:col-span-12 flex items-end justify-end gap-3">
                    <button
                      type="button"
                      onClick={() => { setShowForm(false); resetForm() }}
                      className="px-6 py-3 border border-border text-muted-foreground font-bold rounded-lg hover:bg-card transition-colors text-sm"
                    >
                      Cancel
                    </button>
                    <button
                      type="button"
                      onClick={handleCreate}
                      disabled={submitting}
                      className="px-8 py-3 bg-[#00D4AA] text-background font-bold rounded-lg hover:bg-[#00f5c4] transition-colors text-sm disabled:opacity-50 shadow-lg shadow-[#00D4AA]/20"
                    >
                      {submitting ? "Saving..." : "Save Policy"}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </section>
        )}

        {/* Policy list header */}
        {data && (
          <div className="flex items-center justify-between">
            <h2 className="font-headline text-lg font-bold text-foreground/80 flex items-center gap-2">
              Active Policies
              <span className="text-xs bg-card text-muted-foreground px-2 py-0.5 rounded-full font-normal border border-border">
                {data.total_policies} Configured
              </span>
            </h2>
          </div>
        )}

        {/* Loading state */}
        {!data && !error && (
          <div className="text-center text-muted-foreground py-12 text-sm">
            Loading policies...
          </div>
        )}

        {/* Policy cards */}
        <div className="grid grid-cols-1 gap-4">
          {data && data.list.length > 0 ? (
            data.list.map((p: Policy) => {
              const config = actionConfig[p.action] || actionConfig.BLOCK
              const domains = p.domains || []
              const isExpanded = expanded === p.id

              return (
                <div key={p.id} className="group bg-card border border-border rounded-xl hover:border-[#00D4AA]/40 transition-all duration-300">
                  <div className="p-5 flex items-center justify-between">
                    <div
                      className="flex items-center gap-5 flex-1 cursor-pointer"
                      onClick={() => setExpanded(isExpanded ? null : p.id)}
                    >
                      <div className={`flex items-center justify-center w-12 h-12 rounded-lg border ${config.icon}`}>
                        <ActionIcon action={p.action} />
                      </div>
                      <div>
                        <h3 className="font-bold text-foreground group-hover:text-[#00D4AA] transition-colors">
                          {p.name}
                        </h3>
                        <div className="flex items-center gap-3 mt-1">
                          <span className={`text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider ${config.badge}`}>
                            {p.action}
                          </span>
                          <span className="text-muted-foreground text-xs flex items-center gap-1">
                            <Globe className="w-3.5 h-3.5" />
                            {domains.length} {domains.length === 1 ? "Domain" : "Domains"}
                          </span>
                          {p.description && (
                            <span className="text-muted-foreground text-xs hidden sm:inline">
                              {p.description}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-8">
                      <div className="text-right hidden sm:block">
                        <p className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Priority</p>
                        <p className="font-mono text-foreground">{p.priority}</p>
                      </div>
                      <button
                        onClick={() => handleDelete(p.id)}
                        disabled={deleting === p.id}
                        className="w-9 h-9 flex items-center justify-center text-muted-foreground hover:text-red-500 hover:bg-red-500/10 rounded-lg transition-all disabled:opacity-50"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  {/* Expanded domain list */}
                  {isExpanded && domains.length > 0 && (
                    <div className="px-5 pb-5 pt-0">
                      <div className="border-t border-border pt-4">
                        <p className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold mb-2">Domains</p>
                        <div className="flex flex-wrap gap-2">
                          {domains.map((d) => (
                            <span
                              key={d}
                              className="font-mono text-xs bg-background border border-border rounded-md px-2.5 py-1 text-foreground/80"
                            >
                              {d}
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )
            })
          ) : data ? (
            <div className="text-center text-muted-foreground py-12 text-sm bg-card border border-border rounded-xl">
              No policies configured
            </div>
          ) : null}
        </div>
      </div>
    </>
  )
}
