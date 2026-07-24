"use client"

import { useEffect, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Drawer, DrawerClose, DrawerContent, DrawerDescription,
  DrawerFooter, DrawerHeader, DrawerTitle, DrawerTrigger,
} from "@/components/ui/drawer"
import { Switch } from "@/components/ui/switch"
import {
  getBlocklists, createBlocklist, deleteBlocklist, toggleBlocklist,
  getCategories, toggleCategory,
} from "@/lib/api"
import type { Blocklist, BlocklistListData, Category } from "@/lib/types"
import {
  Plus, Trash2, ShieldCheck, List, RefreshCw, Layers,
} from "lucide-react"

function timeAgo(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000)
  if (seconds < 60) return "just now"
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

export default function BlocklistsPage() {
  const [data, setData] = useState<BlocklistListData | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [showForm, setShowForm] = useState(false)
  const [deleting, setDeleting] = useState<string | null>(null)
  const [toggling, setToggling] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  // Form state
  const [formName, setFormName] = useState("")
  const [formUrl, setFormUrl] = useState("")
  const [formFormat, setFormFormat] = useState("hosts")

  const [error, setError] = useState<string | null>(null)

  const fetchData = () => {
    getBlocklists().then((d) => { setData(d); setError(null) }).catch((e) => setError(e.message))
    getCategories().then(setCategories).catch(() => {})
  }

  useEffect(() => { fetchData() }, [])

  const handleToggleSource = async (bl: Blocklist) => {
    setToggling(bl.id)
    try {
      await toggleBlocklist(bl.id, !bl.enabled)
      fetchData()
    } finally {
      setToggling(null)
    }
  }

  const handleToggleCategory = async (cat: Category) => {
    setToggling(cat.id)
    // Optimistic flip so the switch feels instant.
    setCategories((prev) =>
      prev.map((c) => (c.id === cat.id ? { ...c, enabled: !c.enabled } : c)),
    )
    try {
      await toggleCategory(cat.id, !cat.enabled)
      fetchData()
    } catch {
      // Revert on failure.
      setCategories((prev) =>
        prev.map((c) => (c.id === cat.id ? { ...c, enabled: cat.enabled } : c)),
      )
    } finally {
      setToggling(null)
    }
  }

  const resetForm = () => {
    setFormName("")
    setFormUrl("")
    setFormFormat("hosts")
    setFormError(null)
  }

  const handleCreate = async () => {
    if (!formName || !formUrl) {
      setFormError("Name and URL are required")
      return
    }
    const id = formName.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
    setSubmitting(true)
    setFormError(null)
    try {
      await createBlocklist({
        id,
        name: formName,
        url: formUrl,
        format: formFormat,
      })
      resetForm()
      setShowForm(false)
      fetchData()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : "Failed to create blocklist")
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    setDeleting(id)
    try {
      await deleteBlocklist(id)
      fetchData()
    } finally {
      setDeleting(null)
    }
  }

  const activeSources = data?.active_lists?.filter((bl) => bl.enabled).length ?? 0

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
                  <BreadcrumbLink href="/dashboard">Dashboard</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>Blocklists</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <Drawer direction="right" open={showForm} onOpenChange={(open) => { setShowForm(open); if (!open) resetForm() }}>
          <DrawerTrigger asChild>
            <Button
              size="sm"
              className="bg-[#00D4AA] hover:bg-[#00BD98] text-[#1A1D23] font-semibold gap-2"
            >
              <Plus className="h-4 w-4" />
              Add Source
            </Button>
          </DrawerTrigger>
          <DrawerContent className="data-[vaul-drawer-direction=right]:sm:max-w-md">
            <DrawerHeader>
              <DrawerTitle className="font-headline text-lg">Add Blocklist Source</DrawerTitle>
              <DrawerDescription>Add a URL to fetch blocked domains from</DrawerDescription>
            </DrawerHeader>

            <div className="flex-1 overflow-y-auto px-4">
              {formError && (
                <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
                  {formError}
                </div>
              )}
              <div className="grid gap-5">
                <div className="space-y-2">
                  <Label htmlFor="bl-name" className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                    Name
                  </Label>
                  <Input
                    id="bl-name"
                    placeholder="e.g. StevenBlack Hosts"
                    value={formName}
                    onChange={(e) => setFormName(e.target.value)}
                    className="bg-background border-border rounded-lg"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bl-url" className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                    URL
                  </Label>
                  <Input
                    id="bl-url"
                    placeholder="https://raw.githubusercontent.com/..."
                    value={formUrl}
                    onChange={(e) => setFormUrl(e.target.value)}
                    className="bg-background border-border rounded-lg font-mono text-sm"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bl-format" className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                    Format
                  </Label>
                  <Select value={formFormat} onValueChange={setFormFormat}>
                    <SelectTrigger id="bl-format" className="bg-background border-border rounded-lg">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="hosts">Hosts</SelectItem>
                      <SelectItem value="domains">Domains</SelectItem>
                      <SelectItem value="adblock">Adblock</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>

            <DrawerFooter className="flex-row justify-end gap-3">
              <DrawerClose asChild>
                <Button variant="ghost">Cancel</Button>
              </DrawerClose>
              <Button
                className="bg-[#00D4AA] hover:bg-[#00BD98] text-[#1A1D23] font-semibold"
                onClick={handleCreate}
                disabled={submitting}
              >
                {submitting ? "Adding..." : "Save Source"}
              </Button>
            </DrawerFooter>
          </DrawerContent>
        </Drawer>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        {/* Page title */}
        <div>
          <h2 className="font-headline text-3xl font-bold tracking-tight">Blocklists</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Manage external domain lists to filter network traffic and block threats.
          </p>
        </div>

        {/* Error state */}
        {error && (
          <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}

        {/* Stats summary cards */}
        {data && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {/* Total blocked domains */}
            <div className="bg-card border border-border rounded-xl p-5 flex items-center gap-4 hover:border-[#00D4AA]/30 transition-colors">
              <div className="w-12 h-12 rounded-lg bg-[#00D4AA]/10 flex items-center justify-center text-[#00D4AA]">
                <ShieldCheck className="h-6 w-6" />
              </div>
              <div>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Total blocked domains
                </p>
                <h3 className="text-2xl font-bold font-headline mt-0.5">
                  {data.total_domains.toLocaleString()}
                </h3>
              </div>
            </div>

            {/* Active sources */}
            <div className="bg-card border border-border rounded-xl p-5 flex items-center gap-4 hover:border-[#0EA5E9]/30 transition-colors">
              <div className="w-12 h-12 rounded-lg bg-[#0EA5E9]/10 flex items-center justify-center text-[#0EA5E9]">
                <List className="h-6 w-6" />
              </div>
              <div>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  {activeSources} active source{activeSources !== 1 ? "s" : ""}
                </p>
                <h3 className="text-2xl font-bold font-headline mt-0.5">
                  Global Coverage
                </h3>
              </div>
            </div>

            {/* Last updated */}
            <div className="bg-card border border-border rounded-xl p-5 flex items-center gap-4 hover:border-[#F59E0B]/30 transition-colors">
              <div className="w-12 h-12 rounded-lg bg-[#F59E0B]/10 flex items-center justify-center text-[#F59E0B]">
                <RefreshCw className="h-6 w-6" />
              </div>
              <div>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Auto-refresh interval
                </p>
                <h3 className="text-2xl font-bold font-headline mt-0.5">
                  Every 6h
                </h3>
              </div>
            </div>
          </div>
        )}

        {/* Source list */}
        <div className="space-y-3">
          <div className="flex items-center justify-between px-1 mb-1">
            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
              Active Source Repositories
            </h4>
            {data && (
              <span className="text-xs text-muted-foreground">
                {data.total_blocklists} source{data.total_blocklists !== 1 ? "s" : ""}
              </span>
            )}
          </div>

          {data && data.active_lists.length > 0 ? (
            data.active_lists.map((bl: Blocklist) => (
              <div
                key={bl.id}
                className="bg-card border border-border rounded-xl p-5 flex flex-wrap lg:flex-nowrap items-center gap-6 group hover:shadow-lg hover:border-[#00D4AA]/20 transition-all"
              >
                {/* Name + URL + status dot */}
                <div className="flex items-center gap-4 flex-1 min-w-[260px]">
                  <div className="w-10 h-10 rounded-full bg-green-500/10 flex items-center justify-center shrink-0">
                    <span
                      className={`w-2.5 h-2.5 rounded-full ${
                        bl.enabled ? "bg-green-500 animate-pulse" : "bg-gray-500"
                      }`}
                    />
                  </div>
                  <div className="overflow-hidden">
                    <h4 className="font-headline font-semibold text-lg group-hover:text-[#00D4AA] transition-colors">
                      {bl.name}
                    </h4>
                    <p className="font-mono text-xs text-muted-foreground truncate mt-0.5">
                      {bl.url}
                    </p>
                  </div>
                </div>

                {/* Metadata columns */}
                <div className="flex items-center gap-10 text-sm">
                  <div className="hidden md:block">
                    <p className="text-muted-foreground text-[10px] uppercase font-bold tracking-tight mb-1">
                      Domain Count
                    </p>
                    <p className="text-foreground font-medium">
                      {bl.domains_count.toLocaleString()} domains
                    </p>
                  </div>
                  <div className="hidden sm:block">
                    <p className="text-muted-foreground text-[10px] uppercase font-bold tracking-tight mb-1">
                      Format
                    </p>
                    <span className="inline-block px-2 py-0.5 bg-border rounded-md text-foreground/80 text-[11px] font-mono border border-border">
                      {bl.format}
                    </span>
                  </div>
                  <div className="hidden xl:block">
                    <p className="text-muted-foreground text-[10px] uppercase font-bold tracking-tight mb-1">
                      Last Updated
                    </p>
                    <p className="text-muted-foreground">
                      {bl.updated_at ? timeAgo(bl.updated_at) : "never"}
                    </p>
                  </div>
                  {bl.category && (
                    <div className="hidden lg:block">
                      <p className="text-muted-foreground text-[10px] uppercase font-bold tracking-tight mb-1">
                        Category
                      </p>
                      <span className="inline-block px-2 py-0.5 bg-[#00D4AA]/10 text-[#00D4AA] rounded-md text-[11px] font-medium">
                        {bl.category}
                      </span>
                    </div>
                  )}
                </div>

                {/* Actions */}
                <div className="flex items-center gap-3 ml-auto">
                  <span className={`text-xs font-bold hidden sm:block ${bl.enabled ? "text-green-500" : "text-muted-foreground"}`}>
                    {bl.enabled ? "ACTIVE" : "DISABLED"}
                  </span>
                  <Switch
                    checked={bl.enabled}
                    disabled={toggling === bl.id}
                    onCheckedChange={() => handleToggleSource(bl)}
                    aria-label={`Toggle ${bl.name}`}
                  />
                  <button
                    className="w-9 h-9 rounded-lg hover:bg-destructive/10 hover:text-destructive text-muted-foreground transition-colors flex items-center justify-center disabled:opacity-50"
                    onClick={() => handleDelete(bl.id)}
                    disabled={deleting === bl.id}
                    title="Delete source"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))
          ) : (
            <div className="border-2 border-dashed border-border rounded-xl p-10 flex flex-col items-center justify-center text-center">
              <div className="w-12 h-12 rounded-full bg-border flex items-center justify-center mb-3">
                <ShieldCheck className="h-5 w-5 text-muted-foreground" />
              </div>
              <h5 className="font-headline font-semibold text-foreground">No blocklist sources</h5>
              <p className="text-muted-foreground text-sm mt-1 max-w-xs">
                Add your first blocklist source to start filtering DNS traffic.
              </p>
            </div>
          )}
        </div>

        {/* Curated categories */}
        <div className="space-y-3">
          <div className="flex items-center gap-2 px-1 mb-1">
            <Layers className="h-4 w-4 text-[#00D4AA]" />
            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
              Curated Categories
            </h4>
          </div>
          {categories.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {categories.map((cat: Category) => (
                <div
                  key={cat.id}
                  className="bg-card border border-border rounded-xl p-4 flex items-center gap-4 hover:border-[#00D4AA]/20 transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h5 className="font-headline font-semibold text-foreground truncate">
                        {cat.name}
                      </h5>
                      <span className="text-[10px] text-muted-foreground font-mono shrink-0">
                        {cat.domains_count.toLocaleString()} domains
                      </span>
                    </div>
                    {cat.description && (
                      <p className="text-xs text-muted-foreground truncate mt-0.5">
                        {cat.description}
                      </p>
                    )}
                  </div>
                  <Switch
                    checked={cat.enabled}
                    disabled={toggling === cat.id}
                    onCheckedChange={() => handleToggleCategory(cat)}
                    aria-label={`Toggle ${cat.name}`}
                  />
                </div>
              ))}
            </div>
          ) : (
            <div className="border border-dashed border-border rounded-xl p-6 text-center text-sm text-muted-foreground">
              No curated categories available.
            </div>
          )}
        </div>
      </div>
    </>
  )
}
