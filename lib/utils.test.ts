import { describe, expect, it } from "vitest"

import { cn } from "@/lib/utils"

describe("cn", () => {
  it("joins plain class strings", () => {
    expect(cn("px-2", "text-sm")).toBe("px-2 text-sm")
  })

  it("drops falsy values (clsx conditional handling)", () => {
    const isActive = false
    const isVisible = true
    expect(cn("base", isActive && "active", null, undefined, isVisible && "visible")).toBe(
      "base visible",
    )
  })

  it("supports object and array syntax", () => {
    expect(cn(["flex", "gap-2"], { hidden: false, "rounded-md": true })).toBe(
      "flex gap-2 rounded-md",
    )
  })

  it("resolves conflicting Tailwind utilities so the last one wins (tailwind-merge)", () => {
    expect(cn("px-2", "px-4")).toBe("px-4")
    expect(cn("text-red-500", "text-blue-500")).toBe("text-blue-500")
  })

  it("combines conditional selection with conflict resolution", () => {
    const isLarge = true
    expect(cn("p-2", isLarge && "p-8")).toBe("p-8")
  })

  it("returns an empty string when given no meaningful input", () => {
    expect(cn(false, null, undefined)).toBe("")
  })
})
