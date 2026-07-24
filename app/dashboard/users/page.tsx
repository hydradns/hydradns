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
  DrawerFooter, DrawerHeader, DrawerTitle,
} from "@/components/ui/drawer"
import {
  getUsers, createUser, updateUser, deleteUser,
  getUserTokens, createUserToken, rotateUserToken, revokeUserToken,
} from "@/lib/api"
import type {
  User, UserListData, UserRole, Token,
} from "@/lib/types"
import {
  Plus, Trash2, Users as UsersIcon, ShieldCheck, UserCog, KeyRound,
  Copy, RotateCcw, Ban, Check,
} from "lucide-react"

const ROLES: { value: UserRole; label: string }[] = [
  { value: "admin", label: "Admin" },
  { value: "operator", label: "Operator" },
  { value: "read_only", label: "Read only" },
]

const roleBadge: Record<UserRole, string> = {
  admin: "bg-[#00D4AA]/10 text-[#00D4AA] border border-[#00D4AA]/20",
  operator: "bg-[#0EA5E9]/10 text-[#0EA5E9] border border-[#0EA5E9]/20",
  read_only: "bg-muted text-muted-foreground border border-border",
}

function roleLabel(role: UserRole): string {
  return ROLES.find((r) => r.value === role)?.label ?? role
}

function timeAgo(dateString?: string): string {
  if (!dateString) return "never"
  const date = new Date(dateString)
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000)
  if (seconds < 60) return "just now"
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

export default function UsersPage() {
  const [data, setData] = useState<UserListData | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)

  // Create drawer
  const [createOpen, setCreateOpen] = useState(false)
  const [formUsername, setFormUsername] = useState("")
  const [formPassword, setFormPassword] = useState("")
  const [formRole, setFormRole] = useState<UserRole>("operator")
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  // Edit drawer
  const [editUser, setEditUser] = useState<User | null>(null)
  const [editRole, setEditRole] = useState<UserRole>("operator")
  const [editPassword, setEditPassword] = useState("")
  const [editEnabled, setEditEnabled] = useState(true)
  const [savingEdit, setSavingEdit] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)

  // Token drawer
  const [tokenUser, setTokenUser] = useState<User | null>(null)
  const [tokens, setTokens] = useState<Token[]>([])
  const [tokenName, setTokenName] = useState("")
  const [tokenExpiry, setTokenExpiry] = useState("90")
  const [tokenBusy, setTokenBusy] = useState(false)
  const [tokenError, setTokenError] = useState<string | null>(null)
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const fetchData = () => {
    getUsers().then((d) => { setData(d); setError(null) }).catch((e) => setError(e.message))
  }

  useEffect(() => { fetchData() }, [])

  const fetchTokens = (userId: string) => {
    getUserTokens(userId)
      .then((t) => { setTokens(t); setTokenError(null) })
      .catch((e) => setTokenError(e.message))
  }

  useEffect(() => {
    if (tokenUser) fetchTokens(tokenUser.id)
  }, [tokenUser])

  const openCreate = () => {
    setFormUsername("")
    setFormPassword("")
    setFormRole("operator")
    setCreateError(null)
    setCreateOpen(true)
  }

  const handleCreate = async () => {
    if (!formUsername || !formPassword) {
      setCreateError("Username and password are required")
      return
    }
    setCreating(true)
    setCreateError(null)
    try {
      await createUser({ username: formUsername, password: formPassword, role: formRole })
      setCreateOpen(false)
      fetchData()
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : "Failed to create user")
    } finally {
      setCreating(false)
    }
  }

  const openEdit = (user: User) => {
    setEditUser(user)
    setEditRole(user.role)
    setEditPassword("")
    setEditEnabled(user.enabled)
    setEditError(null)
  }

  const handleEdit = async () => {
    if (!editUser) return
    setSavingEdit(true)
    setEditError(null)
    try {
      await updateUser(editUser.id, {
        role: editRole,
        enabled: editEnabled,
        ...(editPassword ? { password: editPassword } : {}),
      })
      setEditUser(null)
      fetchData()
    } catch (e) {
      setEditError(e instanceof Error ? e.message : "Failed to update user")
    } finally {
      setSavingEdit(false)
    }
  }

  const handleDelete = async (id: string) => {
    setDeleting(id)
    try {
      await deleteUser(id)
      fetchData()
    } finally {
      setDeleting(null)
    }
  }

  const openTokens = (user: User) => {
    setTokens([])
    setTokenName("")
    setTokenExpiry("90")
    setTokenError(null)
    setRevealedSecret(null)
    setTokenUser(user)
  }

  const handleCreateToken = async () => {
    if (!tokenUser || !tokenName) {
      setTokenError("Token name is required")
      return
    }
    setTokenBusy(true)
    setTokenError(null)
    try {
      const res = await createUserToken(tokenUser.id, {
        name: tokenName,
        ...(tokenExpiry !== "never" ? { expires_in_days: parseInt(tokenExpiry) } : {}),
      })
      setRevealedSecret(res.secret)
      setTokenName("")
      fetchTokens(tokenUser.id)
    } catch (e) {
      setTokenError(e instanceof Error ? e.message : "Failed to create token")
    } finally {
      setTokenBusy(false)
    }
  }

  const handleRotateToken = async (tokenId: string) => {
    if (!tokenUser) return
    setTokenBusy(true)
    setTokenError(null)
    try {
      const res = await rotateUserToken(tokenUser.id, tokenId)
      setRevealedSecret(res.secret)
      fetchTokens(tokenUser.id)
    } catch (e) {
      setTokenError(e instanceof Error ? e.message : "Failed to rotate token")
    } finally {
      setTokenBusy(false)
    }
  }

  const handleRevokeToken = async (tokenId: string) => {
    if (!tokenUser) return
    setTokenBusy(true)
    setTokenError(null)
    try {
      await revokeUserToken(tokenUser.id, tokenId)
      fetchTokens(tokenUser.id)
    } catch (e) {
      setTokenError(e instanceof Error ? e.message : "Failed to revoke token")
    } finally {
      setTokenBusy(false)
    }
  }

  const copySecret = async () => {
    if (!revealedSecret || typeof navigator === "undefined" || !navigator.clipboard) return
    await navigator.clipboard.writeText(revealedSecret)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const users = data?.users ?? []
  const adminCount = users.filter((u) => u.role === "admin").length
  const activeCount = users.filter((u) => u.enabled).length

  const labelClass = "text-xs font-bold text-muted-foreground uppercase tracking-widest"

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
                  <BreadcrumbPage>Users</BreadcrumbPage>
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
          Add User
        </Button>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        {/* Page title */}
        <div>
          <h2 className="font-headline text-3xl font-bold tracking-tight">Users &amp; Access</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Manage operator accounts, roles, and API tokens for the gateway.
          </p>
        </div>

        {/* Error state */}
        {error && (
          <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}

        {/* Stats */}
        {data && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-card border border-border rounded-xl p-5 flex items-center gap-4 hover:border-[#00D4AA]/30 transition-colors">
              <div className="w-12 h-12 rounded-lg bg-[#00D4AA]/10 flex items-center justify-center text-[#00D4AA]">
                <UsersIcon className="h-6 w-6" />
              </div>
              <div>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Total users</p>
                <h3 className="text-2xl font-bold font-headline mt-0.5">{data.total_users}</h3>
              </div>
            </div>
            <div className="bg-card border border-border rounded-xl p-5 flex items-center gap-4 hover:border-[#0EA5E9]/30 transition-colors">
              <div className="w-12 h-12 rounded-lg bg-[#0EA5E9]/10 flex items-center justify-center text-[#0EA5E9]">
                <ShieldCheck className="h-6 w-6" />
              </div>
              <div>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Administrators</p>
                <h3 className="text-2xl font-bold font-headline mt-0.5">{adminCount}</h3>
              </div>
            </div>
            <div className="bg-card border border-border rounded-xl p-5 flex items-center gap-4 hover:border-[#F59E0B]/30 transition-colors">
              <div className="w-12 h-12 rounded-lg bg-[#F59E0B]/10 flex items-center justify-center text-[#F59E0B]">
                <UserCog className="h-6 w-6" />
              </div>
              <div>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Active accounts</p>
                <h3 className="text-2xl font-bold font-headline mt-0.5">{activeCount}</h3>
              </div>
            </div>
          </div>
        )}

        {/* User list */}
        <div className="space-y-3">
          <div className="flex items-center justify-between px-1 mb-1">
            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">Accounts</h4>
            {data && (
              <span className="text-xs text-muted-foreground">
                {data.total_users} user{data.total_users !== 1 ? "s" : ""}
              </span>
            )}
          </div>

          {data && users.length > 0 ? (
            users.map((u) => (
              <div
                key={u.id}
                className="bg-card border border-border rounded-xl p-5 flex flex-wrap lg:flex-nowrap items-center gap-6 group hover:shadow-lg hover:border-[#00D4AA]/20 transition-all"
              >
                <div className="flex items-center gap-4 flex-1 min-w-[220px]">
                  <div className="w-10 h-10 rounded-full bg-[#00D4AA]/10 flex items-center justify-center shrink-0 text-[#00D4AA] font-headline font-bold uppercase">
                    {u.username.charAt(0)}
                  </div>
                  <div className="overflow-hidden">
                    <h4 className="font-headline font-semibold text-lg group-hover:text-[#00D4AA] transition-colors">
                      {u.username}
                    </h4>
                    <p className="font-mono text-xs text-muted-foreground truncate mt-0.5">
                      Last login {timeAgo(u.last_login_at)}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-10 text-sm">
                  <div>
                    <p className="text-muted-foreground text-[10px] uppercase font-bold tracking-tight mb-1">Role</p>
                    <span className={`inline-block px-2 py-0.5 rounded-md text-[11px] font-medium ${roleBadge[u.role]}`}>
                      {roleLabel(u.role)}
                    </span>
                  </div>
                  <div className="hidden sm:block">
                    <p className="text-muted-foreground text-[10px] uppercase font-bold tracking-tight mb-1">Status</p>
                    <span className={`text-xs font-bold ${u.enabled ? "text-green-500" : "text-muted-foreground"}`}>
                      {u.enabled ? "ACTIVE" : "DISABLED"}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-1 ml-auto">
                  <button
                    className="h-9 px-3 rounded-lg hover:bg-[#00D4AA]/10 hover:text-[#00D4AA] text-muted-foreground transition-colors flex items-center gap-1.5 text-xs font-semibold"
                    onClick={() => openTokens(u)}
                    title="Manage tokens"
                  >
                    <KeyRound className="h-4 w-4" />
                    Tokens
                  </button>
                  <button
                    className="w-9 h-9 rounded-lg hover:bg-[#0EA5E9]/10 hover:text-[#0EA5E9] text-muted-foreground transition-colors flex items-center justify-center"
                    onClick={() => openEdit(u)}
                    title="Edit user"
                  >
                    <UserCog className="h-4 w-4" />
                  </button>
                  <button
                    className="w-9 h-9 rounded-lg hover:bg-destructive/10 hover:text-destructive text-muted-foreground transition-colors flex items-center justify-center disabled:opacity-50"
                    onClick={() => handleDelete(u.id)}
                    disabled={deleting === u.id}
                    title="Delete user"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))
          ) : data ? (
            <div className="border-2 border-dashed border-border rounded-xl p-10 flex flex-col items-center justify-center text-center">
              <div className="w-12 h-12 rounded-full bg-border flex items-center justify-center mb-3">
                <UsersIcon className="h-5 w-5 text-muted-foreground" />
              </div>
              <h5 className="font-headline font-semibold text-foreground">No users yet</h5>
              <p className="text-muted-foreground text-sm mt-1 max-w-xs">
                Create your first operator account to delegate gateway access.
              </p>
            </div>
          ) : (
            <div className="text-center text-muted-foreground py-12 text-sm">Loading users...</div>
          )}
        </div>
      </div>

      {/* Create drawer */}
      <Drawer direction="right" open={createOpen} onOpenChange={setCreateOpen}>
        <DrawerContent className="data-[vaul-drawer-direction=right]:sm:max-w-md">
          <DrawerHeader>
            <DrawerTitle className="font-headline text-lg">Add User</DrawerTitle>
            <DrawerDescription>Create a new account with a role.</DrawerDescription>
          </DrawerHeader>
          <div className="flex-1 overflow-y-auto px-4">
            {createError && (
              <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
                {createError}
              </div>
            )}
            <div className="grid gap-5">
              <div className="space-y-2">
                <Label htmlFor="u-name" className={labelClass}>Username</Label>
                <Input
                  id="u-name"
                  placeholder="e.g. jane.doe"
                  value={formUsername}
                  onChange={(e) => setFormUsername(e.target.value)}
                  className="bg-background border-border rounded-lg"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="u-pass" className={labelClass}>Password</Label>
                <Input
                  id="u-pass"
                  type="password"
                  placeholder="Set an initial password"
                  value={formPassword}
                  onChange={(e) => setFormPassword(e.target.value)}
                  className="bg-background border-border rounded-lg"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="u-role" className={labelClass}>Role</Label>
                <Select value={formRole} onValueChange={(v) => setFormRole(v as UserRole)}>
                  <SelectTrigger id="u-role" className="bg-background border-border rounded-lg">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ROLES.map((r) => (
                      <SelectItem key={r.value} value={r.value}>{r.label}</SelectItem>
                    ))}
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
              disabled={creating}
            >
              {creating ? "Creating..." : "Create User"}
            </Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

      {/* Edit drawer */}
      <Drawer direction="right" open={!!editUser} onOpenChange={(open) => { if (!open) setEditUser(null) }}>
        <DrawerContent className="data-[vaul-drawer-direction=right]:sm:max-w-md">
          <DrawerHeader>
            <DrawerTitle className="font-headline text-lg">Edit {editUser?.username}</DrawerTitle>
            <DrawerDescription>Update role, password, or account status.</DrawerDescription>
          </DrawerHeader>
          <div className="flex-1 overflow-y-auto px-4">
            {editError && (
              <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
                {editError}
              </div>
            )}
            <div className="grid gap-5">
              <div className="space-y-2">
                <Label htmlFor="e-role" className={labelClass}>Role</Label>
                <Select value={editRole} onValueChange={(v) => setEditRole(v as UserRole)}>
                  <SelectTrigger id="e-role" className="bg-background border-border rounded-lg">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ROLES.map((r) => (
                      <SelectItem key={r.value} value={r.value}>{r.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="e-pass" className={labelClass}>New password</Label>
                <Input
                  id="e-pass"
                  type="password"
                  placeholder="Leave blank to keep current"
                  value={editPassword}
                  onChange={(e) => setEditPassword(e.target.value)}
                  className="bg-background border-border rounded-lg"
                />
              </div>
              <div className="flex items-center justify-between rounded-lg border border-border bg-background p-4">
                <div>
                  <p className="text-sm font-semibold text-foreground">Account enabled</p>
                  <p className="text-xs text-muted-foreground">Disabled users cannot sign in.</p>
                </div>
                <button
                  type="button"
                  onClick={() => setEditEnabled((v) => !v)}
                  className={`relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors ${editEnabled ? "bg-[#00D4AA]" : "bg-muted"}`}
                  role="switch"
                  aria-checked={editEnabled}
                  aria-label="Toggle account enabled"
                >
                  <span className={`inline-block h-5 w-5 rounded-full bg-white shadow transition-transform ${editEnabled ? "translate-x-6" : "translate-x-1"}`} />
                </button>
              </div>
            </div>
          </div>
          <DrawerFooter className="flex-row justify-end gap-3">
            <DrawerClose asChild>
              <Button variant="ghost">Cancel</Button>
            </DrawerClose>
            <Button
              className="bg-[#00D4AA] hover:bg-[#00BD98] text-[#1A1D23] font-semibold"
              onClick={handleEdit}
              disabled={savingEdit}
            >
              {savingEdit ? "Saving..." : "Save Changes"}
            </Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>

      {/* Token management drawer */}
      <Drawer direction="right" open={!!tokenUser} onOpenChange={(open) => { if (!open) setTokenUser(null) }}>
        <DrawerContent className="data-[vaul-drawer-direction=right]:sm:max-w-md">
          <DrawerHeader>
            <DrawerTitle className="font-headline text-lg flex items-center gap-2">
              <KeyRound className="h-4 w-4 text-[#00D4AA]" />
              API Tokens — {tokenUser?.username}
            </DrawerTitle>
            <DrawerDescription>Tokens inherit this user&apos;s role. Secrets are shown only once.</DrawerDescription>
          </DrawerHeader>
          <div className="flex-1 overflow-y-auto px-4 space-y-5">
            {tokenError && (
              <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
                {tokenError}
              </div>
            )}

            {/* Revealed secret (once) */}
            {revealedSecret && (
              <div className="rounded-lg border border-[#00D4AA]/40 bg-[#00D4AA]/10 p-3 space-y-2">
                <p className="text-[10px] font-bold text-[#00D4AA] uppercase tracking-widest">
                  Copy this token now — it won&apos;t be shown again
                </p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 font-mono text-xs break-all text-foreground">{revealedSecret}</code>
                  <button
                    onClick={copySecret}
                    className="w-8 h-8 rounded-md bg-background border border-border flex items-center justify-center hover:text-[#00D4AA] transition-colors shrink-0"
                    title="Copy token"
                  >
                    {copied ? <Check className="h-4 w-4 text-[#00D4AA]" /> : <Copy className="h-4 w-4" />}
                  </button>
                </div>
              </div>
            )}

            {/* Create token form */}
            <div className="rounded-lg border border-border bg-background p-4 space-y-4">
              <p className="text-xs font-bold text-muted-foreground uppercase tracking-widest">New token</p>
              <div className="space-y-2">
                <Label htmlFor="t-name" className={labelClass}>Name</Label>
                <Input
                  id="t-name"
                  placeholder="e.g. ci-pipeline"
                  value={tokenName}
                  onChange={(e) => setTokenName(e.target.value)}
                  className="bg-card border-border rounded-lg"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="t-expiry" className={labelClass}>Expires</Label>
                <Select value={tokenExpiry} onValueChange={setTokenExpiry}>
                  <SelectTrigger id="t-expiry" className="bg-card border-border rounded-lg">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="30">30 days</SelectItem>
                    <SelectItem value="60">60 days</SelectItem>
                    <SelectItem value="90">90 days</SelectItem>
                    <SelectItem value="365">1 year</SelectItem>
                    <SelectItem value="never">Never</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button
                className="w-full bg-[#00D4AA] hover:bg-[#00BD98] text-[#1A1D23] font-semibold gap-2"
                onClick={handleCreateToken}
                disabled={tokenBusy}
              >
                <Plus className="h-4 w-4" />
                Create Token
              </Button>
            </div>

            {/* Token list */}
            <div className="space-y-2">
              <p className="text-xs font-bold text-muted-foreground uppercase tracking-widest px-1">Existing tokens</p>
              {tokens.length > 0 ? (
                tokens.map((t) => (
                  <div key={t.id} className="rounded-lg border border-border bg-card p-3 flex items-center gap-3">
                    <div className="flex-1 overflow-hidden">
                      <div className="flex items-center gap-2">
                        <span className="font-headline font-semibold text-sm truncate">{t.name}</span>
                        {t.revoked && (
                          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-destructive/10 text-destructive uppercase">Revoked</span>
                        )}
                      </div>
                      <p className="font-mono text-[11px] text-muted-foreground mt-0.5 truncate">
                        {t.prefix}••• · used {timeAgo(t.last_used_at)}
                      </p>
                    </div>
                    <button
                      onClick={() => handleRotateToken(t.id)}
                      disabled={tokenBusy || t.revoked}
                      className="w-8 h-8 rounded-md flex items-center justify-center text-muted-foreground hover:text-[#0EA5E9] hover:bg-[#0EA5E9]/10 transition-colors disabled:opacity-40"
                      title="Rotate token"
                    >
                      <RotateCcw className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleRevokeToken(t.id)}
                      disabled={tokenBusy || t.revoked}
                      className="w-8 h-8 rounded-md flex items-center justify-center text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-40"
                      title="Revoke token"
                    >
                      <Ban className="h-4 w-4" />
                    </button>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground px-1 py-4">No tokens for this user yet.</p>
              )}
            </div>
          </div>
          <DrawerFooter className="flex-row justify-end">
            <DrawerClose asChild>
              <Button variant="ghost">Close</Button>
            </DrawerClose>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>
    </>
  )
}
