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
import { getPolicies, createPolicy, deletePolicy } from "@/lib/api"
import type { Policy, PolicyListData } from "@/lib/types"

const actionColors: Record<string, string> = {
  BLOCK: "destructive",
  ALLOW: "default",
  REDIRECT: "secondary",
}

export default function PoliciesPage() {
  const [data, setData] = useState<PolicyListData | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [deleting, setDeleting] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

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
                  <BreadcrumbPage>Policies</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <Button size="sm" onClick={() => setShowForm(!showForm)}>
          {showForm ? "Cancel" : "Add Policy"}
        </Button>
      </header>

      <div className="flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}
        {data && (
          <div className="flex gap-4">
            <Badge variant="outline">{data.total_policies} total</Badge>
            <Badge variant="default">{data.active_policies} active</Badge>
            <Badge variant="secondary">{data.inactive_policies} inactive</Badge>
          </div>
        )}

        {/* Add Form */}
        {showForm && (
          <Card>
            <CardHeader>
              <CardTitle>Create Policy</CardTitle>
              <CardDescription>Define rules for blocking, allowing, or redirecting DNS queries</CardDescription>
            </CardHeader>
            <CardContent>
              {formError && (
                <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
                  {formError}
                </div>
              )}
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="pol-id">ID</Label>
                  <Input id="pol-id" placeholder="e.g. block-social" value={formId} onChange={(e) => setFormId(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="pol-name">Name</Label>
                  <Input id="pol-name" placeholder="e.g. Block Social Media" value={formName} onChange={(e) => setFormName(e.target.value)} />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <Label htmlFor="pol-desc">Description</Label>
                  <Input id="pol-desc" placeholder="Optional description" value={formDescription} onChange={(e) => setFormDescription(e.target.value)} />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <Label htmlFor="pol-domains">Domains (comma or newline separated)</Label>
                  <textarea
                    id="pol-domains"
                    className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] font-mono"
                    placeholder={"facebook.com\ninstagram.com\ntiktok.com"}
                    value={formDomains}
                    onChange={(e) => setFormDomains(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="pol-action">Action</Label>
                  <Select value={formAction} onValueChange={setFormAction}>
                    <SelectTrigger id="pol-action">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="BLOCK">Block</SelectItem>
                      <SelectItem value="ALLOW">Allow</SelectItem>
                      <SelectItem value="REDIRECT">Redirect</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="pol-priority">Priority</Label>
                  <Input id="pol-priority" type="number" placeholder="100" value={formPriority} onChange={(e) => setFormPriority(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="pol-category">Category</Label>
                  <Input id="pol-category" placeholder="e.g. social, security" value={formCategory} onChange={(e) => setFormCategory(e.target.value)} />
                </div>
              </div>
              <div className="mt-4 flex justify-end">
                <Button onClick={handleCreate} disabled={submitting}>
                  {submitting ? "Creating..." : "Create Policy"}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Table */}
        <Card>
          <CardHeader>
            <CardTitle>Policies</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>Domains</TableHead>
                  <TableHead>Category</TableHead>
                  <TableHead className="text-right">Priority</TableHead>
                  <TableHead></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data && data.list.length > 0 ? (
                  data.list.map((p: Policy) => (
                    <TableRow key={p.id}>
                      <TableCell>
                        <div className="font-medium">{p.name}</div>
                        {p.description && (
                          <div className="text-xs text-muted-foreground">{p.description}</div>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant={(actionColors[p.action] as "destructive" | "default" | "secondary") || "outline"}>
                          {p.action}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {(p.domains || []).slice(0, 3).map((d) => (
                            <Badge key={d} variant="outline" className="font-mono text-xs">
                              {d}
                            </Badge>
                          ))}
                          {(p.domains || []).length > 3 && (
                            <Badge variant="outline" className="text-xs">
                              +{p.domains.length - 3} more
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{p.category || "—"}</TableCell>
                      <TableCell className="text-right tabular-nums">{p.priority}</TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleDelete(p.id)}
                          disabled={deleting === p.id}
                        >
                          {deleting === p.id ? "..." : "Delete"}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      No policies configured
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
