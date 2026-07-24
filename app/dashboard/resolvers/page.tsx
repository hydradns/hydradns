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
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Drawer, DrawerClose, DrawerContent, DrawerDescription,
  DrawerFooter, DrawerHeader, DrawerTitle,
} from "@/components/ui/drawer"
import {
  getResolvers, createResolver, updateResolver, deleteResolver,
} from "@/lib/api"
import type { Resolver } from "@/lib/types"
import { Plus, Pencil, Trash2, Server } from "lucide-react"

export default function ResolversPage() {
  const [resolvers, setResolvers] = useState<Resolver[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)

  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  // Form state
  const [formName, setFormName] = useState("")
  const [formAddress, setFormAddress] = useState("")
  const [formProtocol, setFormProtocol] = useState("udp")

  const fetchData = () => {
    getResolvers()
      .then((r) => { setResolvers(r); setError(null) })
      .catch((e) => setError(e.message))
      .finally(() => setLoaded(true))
  }

  useEffect(() => { fetchData() }, [])

  const resetForm = () => {
    setFormName("")
    setFormAddress("")
    setFormProtocol("udp")
    setFormError(null)
    setEditingId(null)
  }

  const openCreate = () => {
    resetForm()
    setShowForm(true)
  }

  const openEdit = (r: Resolver) => {
    setEditingId(r.id)
    setFormName(r.name)
    setFormAddress(r.address)
    setFormProtocol(r.protocol || "udp")
    setFormError(null)
    setShowForm(true)
  }

  const handleSubmit = async () => {
    if (!formName || !formAddress) {
      setFormError("Name and address are required")
      return
    }
    setSubmitting(true)
    setFormError(null)
    const payload = { name: formName, address: formAddress, protocol: formProtocol }
    try {
      if (editingId) {
        await updateResolver(editingId, payload)
      } else {
        const id = formName.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "")
        await createResolver({ id, ...payload })
      }
      resetForm()
      setShowForm(false)
      fetchData()
    } catch (e) {
      setFormError(e instanceof Error ? e.message : "Failed to save resolver")
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    setDeleting(id)
    try {
      await deleteResolver(id)
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
                  <BreadcrumbLink href="/dashboard">Dashboard</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>Resolvers</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>

        <Button
          size="sm"
          onClick={openCreate}
          className="bg-[#00D4AA] hover:bg-[#00BD98] text-[#1A1D23] font-semibold gap-2"
        >
          <Plus className="h-4 w-4" />
          Add Resolver
        </Button>

        <Drawer direction="right" open={showForm} onOpenChange={(open) => { setShowForm(open); if (!open) resetForm() }}>
          <DrawerContent className="data-[vaul-drawer-direction=right]:sm:max-w-md">
            <DrawerHeader>
              <DrawerTitle className="font-headline text-lg">
                {editingId ? "Edit Resolver" : "Add Resolver"}
              </DrawerTitle>
              <DrawerDescription>
                {editingId
                  ? "Update this upstream resolver"
                  : "Add an upstream resolver to forward queries to"}
              </DrawerDescription>
            </DrawerHeader>

            <div className="flex-1 overflow-y-auto px-4">
              {formError && (
                <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
                  {formError}
                </div>
              )}
              <div className="grid gap-5">
                <div className="space-y-2">
                  <Label htmlFor="r-name" className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                    Name
                  </Label>
                  <Input
                    id="r-name"
                    placeholder="e.g. Cloudflare Primary"
                    value={formName}
                    onChange={(e) => setFormName(e.target.value)}
                    className="bg-background border-border rounded-lg"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="r-address" className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                    Address
                  </Label>
                  <Input
                    id="r-address"
                    placeholder="e.g. 1.1.1.1:53"
                    value={formAddress}
                    onChange={(e) => setFormAddress(e.target.value)}
                    className="bg-background border-border rounded-lg font-mono text-sm"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="r-protocol" className="text-xs font-bold text-muted-foreground uppercase tracking-widest">
                    Protocol
                  </Label>
                  <Select value={formProtocol} onValueChange={setFormProtocol}>
                    <SelectTrigger id="r-protocol" className="bg-background border-border rounded-lg">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="udp">UDP</SelectItem>
                      <SelectItem value="tcp">TCP</SelectItem>
                      <SelectItem value="dot">DoT (DNS-over-TLS)</SelectItem>
                      <SelectItem value="doh">DoH (DNS-over-HTTPS)</SelectItem>
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
                onClick={handleSubmit}
                disabled={submitting}
              >
                {submitting ? "Saving..." : editingId ? "Update Resolver" : "Save Resolver"}
              </Button>
            </DrawerFooter>
          </DrawerContent>
        </Drawer>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        <div>
          <h2 className="font-headline text-3xl font-bold tracking-tight">Resolvers</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Manage the upstream DNS resolvers HydraDNS forwards allowed queries to.
          </p>
        </div>

        {error && (
          <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}

        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="flex items-center gap-2 border-b border-border px-6 py-4">
            <Server className="h-5 w-5 text-[#00D4AA]" />
            <h3 className="font-headline font-bold text-foreground">Upstream Resolvers</h3>
            <span className="ml-auto text-xs text-muted-foreground">
              {resolvers.length} configured
            </span>
          </div>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Name</TableHead>
                  <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Address</TableHead>
                  <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Protocol</TableHead>
                  <TableHead className="px-6 text-right text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {resolvers.length > 0 ? (
                  resolvers.map((r) => (
                    <TableRow key={r.id} className="hover:bg-background/30">
                      <TableCell className="px-6 py-4 text-sm font-medium text-foreground">
                        {r.name || r.address}
                      </TableCell>
                      <TableCell className="px-6 py-4 font-mono text-xs text-muted-foreground">
                        {r.address}
                      </TableCell>
                      <TableCell className="px-6 py-4">
                        <Badge variant="outline" className="text-[10px] font-bold uppercase">
                          {r.protocol}
                        </Badge>
                      </TableCell>
                      <TableCell className="px-6 py-4">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            onClick={() => openEdit(r)}
                            title="Edit resolver"
                            className="w-9 h-9 flex items-center justify-center text-muted-foreground hover:text-[#00D4AA] hover:bg-[#00D4AA]/10 rounded-lg transition-all"
                          >
                            <Pencil className="h-4 w-4" />
                          </button>
                          <button
                            onClick={() => handleDelete(r.id)}
                            disabled={deleting === r.id}
                            title="Remove resolver"
                            className="w-9 h-9 flex items-center justify-center text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg transition-all disabled:opacity-50"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={4} className="py-10 text-center text-sm text-muted-foreground">
                      {loaded ? "No resolvers configured" : "Loading resolvers..."}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </>
  )
}
