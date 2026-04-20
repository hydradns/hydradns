import type { Config } from "tailwindcss";

export default {
  darkMode: ["class"],
  content: ["./pages/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./app/**/*.{ts,tsx}", "./src/**/*.{ts,tsx}"],
  prefix: "",
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        // HydraDNS Security Theme — mirrors the dashboard palette
        surface: {
          DEFAULT: "hsl(var(--surface))",
          container: "hsl(var(--surface-container))",
          low: "hsl(var(--surface-container-low))",
          high: "hsl(var(--surface-container-high))",
          highest: "hsl(var(--surface-container-highest))",
          lowest: "hsl(var(--surface-container-lowest))",
        },
        brand: {
          teal: "hsl(var(--brand-teal))",
          "teal-bright": "hsl(var(--brand-teal-bright))",
          sky: "hsl(var(--brand-sky))",
          amber: "hsl(var(--brand-amber))",
          green: "hsl(var(--brand-green))",
          red: "hsl(var(--brand-red))",
        },
        outline: {
          DEFAULT: "hsl(var(--outline))",
          variant: "hsl(var(--outline-variant))",
        },
      },
      fontFamily: {
        headline: ["'Space Grotesk'", "system-ui", "sans-serif"],
        body: ["Inter", "system-ui", "-apple-system", "sans-serif"],
        mono: ["'Fira Code'", "ui-monospace", "SFMono-Regular", "Menlo", "monospace"],
      },
      letterSpacing: {
        tightest: "-0.03em",
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      boxShadow: {
        "glow-primary": "0 0 30px rgba(0, 212, 170, 0.12)",
        "glow-primary-strong": "0 0 60px rgba(0, 212, 170, 0.25)",
        "glow-sky": "0 0 30px rgba(137, 206, 255, 0.12)",
      },
      keyframes: {
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
        float: {
          "0%, 100%": { transform: "translateY(0)" },
          "50%": { transform: "translateY(-10px)" },
        },
        "pulse-ring": {
          "0%": { boxShadow: "0 0 0 0 rgba(0, 212, 170, 0.4)" },
          "70%": { boxShadow: "0 0 0 10px rgba(0, 212, 170, 0)" },
          "100%": { boxShadow: "0 0 0 0 rgba(0, 212, 170, 0)" },
        },
        "gradient-pan": {
          "0%, 100%": { backgroundPosition: "0% 50%" },
          "50%": { backgroundPosition: "100% 50%" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        float: "float 6s ease-in-out infinite",
        "pulse-ring": "pulse-ring 2s infinite",
        "gradient-pan": "gradient-pan 8s ease infinite",
      },
    },
  },
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  plugins: [require("tailwindcss-animate")],
} satisfies Config;
