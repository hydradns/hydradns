// Registers jest-dom matchers (toBeInTheDocument, toHaveClass, ...) on Vitest's
// expect and runs after each test cleanup for React Testing Library.
import "@testing-library/jest-dom/vitest"
import { afterEach } from "vitest"

// Global reset so a token (or anything else) written to storage by one test
// can never leak into the next test file/describe block. Without this, a
// test that calls localStorage.setItem and doesn't clean up after itself
// silently poisons whatever runs next in the same file (see
// lib/api.test.ts's former "attaches a bearer token from localStorage"
// test, which relied on later blocks not asserting on Authorization
// headers rather than on this being enforced).
afterEach(() => {
  if (typeof window !== "undefined") {
    window.localStorage?.clear()
    window.sessionStorage?.clear()
  }
})

// jsdom lacks a few browser APIs that Radix UI / vaul rely on at mount time.
// Provide minimal shims so component tests can render dialogs, drawers, and
// switches without throwing.
if (typeof window !== "undefined") {
  // Node's own built-in `localStorage`/`sessionStorage` globals (stable since
  // Node 22) shadow jsdom's working implementation: Vitest's jsdom environment
  // only forwards window keys that aren't already present as global keys, and
  // Node defines these two unconditionally (returning `undefined` unless
  // `--localstorage-file` is passed). Without this shim every `localStorage.*`
  // call in app code throws `Cannot read properties of undefined`. Replace
  // both with a minimal in-memory Storage implementation.
  const createMemoryStorage = (): Storage => {
    let store = new Map<string, string>()
    return {
      getItem: (key: string) => (store.has(key) ? store.get(key)! : null),
      setItem: (key: string, value: string) => {
        store.set(key, String(value))
      },
      removeItem: (key: string) => {
        store.delete(key)
      },
      clear: () => {
        store = new Map<string, string>()
      },
      key: (index: number) => Array.from(store.keys())[index] ?? null,
      get length() {
        return store.size
      },
    }
  }

  if (!window.localStorage) {
    window.localStorage = createMemoryStorage()
  }
  if (!window.sessionStorage) {
    window.sessionStorage = createMemoryStorage()
  }

  if (!window.matchMedia) {
    window.matchMedia = (query: string) =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList
  }

  if (!window.ResizeObserver) {
    window.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }

  if (!Element.prototype.scrollIntoView) {
    Element.prototype.scrollIntoView = () => {}
  }
  if (!Element.prototype.hasPointerCapture) {
    Element.prototype.hasPointerCapture = () => false
  }
  if (!Element.prototype.setPointerCapture) {
    Element.prototype.setPointerCapture = () => {}
  }
  if (!Element.prototype.releasePointerCapture) {
    Element.prototype.releasePointerCapture = () => {}
  }
}
