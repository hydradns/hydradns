"use client"

import { useEffect, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { getBlocklists, createBlocklist, deleteBlocklist } from "@/lib/api"
import type { Blocklist, BlocklistListData } from "@/lib/types"

export default function BlocklistsPage() {
  const [data, setData] = useState<BlocklistListData | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [deleting, setDeleting] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  // Form state
  const [formId, setFormId] = useState("")
  const [formName, setFormName] = useState("")
  const [formUrl, setFormUrl] = useState("")
  const [formFormat, setFormFormat] = useState("hosts")
  const [formCategory, setFormCategory] = useState("")

  const [error, setError] = useState<string | null>(null)

  const fetchData = () => {
    getBlocklists().then((d) => { setData(d); setError(null) }).catch((e) => setError(e.message))
  }

  useEffect(() => { fetchData() }, [])

  const resetForm = () => {
    setFormId("")
    setFormName("")
    setFormUrl("")
    setFormFormat("hosts")
    setFormCategory("")
    setFormError(null)
  }

  const handleCreate = async () => {
    if (!formId || !formName || !formUrl) {
      setFormError("ID, Name, and URL are required")
      return
    }
    setSubmitting(true)
    setFormError(null)
    try {
      await createBlocklist({
        id: formId,
        name: formName,
        url: formUrl,
        format: formFormat,
        category: formCategory || undefined,
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

  return (
    <>
      <header className="flex flex-wrap gap-3 min-h-20 py-4 shrink-0 items-center border-b">
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
                  <BreadcrumbPage>Blocklists</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <Button size="sm" onClick={() => setShowForm(!showForm)}>
          {showForm ? "Cancel" : "Add Blocklist"}
        </Button>
      </header>

      <div className="flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}
        {/* Stats */}
        {data && (
          <div className="flex gap-4">
            <Badge variant="outline">{data.total_blocklists} sources</Badge>
            <Badge variant="outline">{data.total_domains.toLocaleString()} domains blocked</Badge>
          </div>
        )}

        {/* Add Form */}
        {showForm && (
          <Card>
            <CardHeader>
              <CardTitle>Add Blocklist Source</CardTitle>
              <CardDescription>Add a new blocklist URL to fetch and block domains from</CardDescription>
            </CardHeader>
            <CardContent>
              {formError && (
                <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
                  {formError}
                </div>
              )}
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="bl-id">ID</Label>
                  <Input id="bl-id" placeholder="e.g. steven-black" value={formId} onChange={(e) => setFormId(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bl-name">Name</Label>
                  <Input id="bl-name" placeholder="e.g. StevenBlack Hosts" value={formName} onChange={(e) => setFormName(e.target.value)} />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <Label htmlFor="bl-url">URL</Label>
                  <Input id="bl-url" placeholder="https://raw.githubusercontent.com/..." value={formUrl} onChange={(e) => setFormUrl(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bl-format">Format</Label>
                  <Select value={formFormat} onValueChange={setFormFormat}>
                    <SelectTrigger id="bl-format">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="hosts">Hosts</SelectItem>
                      <SelectItem value="domains">Domains</SelectItem>
                      <SelectItem value="adblock">Adblock</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bl-category">Category</Label>
                  <Input id="bl-category" placeholder="e.g. ads, malware" value={formCategory} onChange={(e) => setFormCategory(e.target.value)} />
                </div>
              </div>
              <div className="mt-4 flex justify-end">
                <Button onClick={handleCreate} disabled={submitting}>
                  {submitting ? "Adding..." : "Add Source"}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Table */}
        <Card>
          <CardHeader>
            <CardTitle>Blocklist Sources</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Category</TableHead>
                  <TableHead>Format</TableHead>
                  <TableHead className="text-right">Domains</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data && data.active_lists.length > 0 ? (
                  data.active_lists.map((bl: Blocklist) => (
                    <TableRow key={bl.id}>
                      <TableCell>
                        <div className="font-medium">{bl.name}</div>
                        <div className="text-xs text-muted-foreground truncate max-w-[300px]">{bl.url}</div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{bl.category || "—"}</Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs">{bl.format}</TableCell>
                      <TableCell className="text-right tabular-nums">{bl.domains_count.toLocaleString()}</TableCell>
                      <TableCell>
                        <Badge variant={bl.enabled ? "default" : "secondary"}>
                          {bl.enabled ? "Active" : "Disabled"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleDelete(bl.id)}
                          disabled={deleting === bl.id}
                        >
                          {deleting === bl.id ? "..." : "Delete"}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      No blocklist sources configured
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
