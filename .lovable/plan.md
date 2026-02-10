

# HydraDNS Landing Page

## Overview
A modern, dark-themed, high-converting landing page for HydraDNS — an open-source DNS security gateway. Inspired by Linear.app and Vercel aesthetics: clean, moody, premium, with glass-morphism cards and subtle animations.

## Design System
- **Dark-only theme** with deep navy background (`#0A0F1C`), electric blue (`#3B82F6`) and cyan (`#06B6D4`) accents, purple (`#8B5CF6`) secondary accent
- **Typography:** Clean sans-serif, large bold headlines, lightweight body text, generous whitespace
- **Cards:** Glass-morphism style with subtle borders and soft glow effects
- **Animations:** Scroll-triggered fade-ins and slide-ups throughout all sections

## Sections (in order)

### 1. Sticky Navigation Bar
- Glass-morphism background, logo (uploaded image) + "HydraDNS" on left
- Links: Features, How It Works, Comparison, Open Source
- Two CTAs: "Get Started" (solid) + "Star on GitHub" (with icon)
- All external links open in new tabs

### 2. Hero Section
- Lightweight canvas particle/network animation in the background (floating dots + connecting lines)
- Bold headline: "Block threats before they ever connect."
- Subheadline describing the product value
- Two CTA buttons linking to GitHub
- Trust line: "Open source · GPL-3.0 · Self-hosted · No data leaves your network"
- Floating glass-morphism dashboard mockup card with hardcoded stats (124,847 queries, 18,293 blocked, 14.6% block rate, mini line chart)

### 3. "The Problem" Section
- "WHY HydraDNS" label, "Your DNS is leaking everything." headline
- Three short paragraphs explaining the DNS privacy problem
- Minimal animated/static diagram: Device → DNS Query → HydraDNS shield → Clean/Blocked paths

### 4. Features Grid (8 cards)
- 3-column responsive grid of glass-morphism feature cards
- Each with icon, title, and 1-2 line description
- Covers: DNS-Layer Blocking, Privacy First, Sub-Millisecond Decisions, Real-Time Dashboard, Flexible Policy Engine, Multiple Blocklist Sources, One Command Deploy, Full REST API

### 5. How It Works (3 steps)
- Three horizontal steps connected by dashed lines (vertical on mobile)
- Deploy → Point Your DNS → Monitor & Control
- Each with icon and short description

### 6. Comparison Table
- HydraDNS vs Pi-hole vs NextDNS vs AdGuard Home
- HydraDNS column visually highlighted with accent border
- 10 feature rows with ✅/❌ and green/red coloring

### 7. Quick Start Code Block
- Terminal-style dark code block with macOS title bar dots
- Shows clone + docker-compose + test commands
- Note below: "That's it. No config files to edit, no dependencies to install."

### 8. Stats / Metrics Strip
- Horizontal bar with 4 key numbers: 17+ API Endpoints, <1ms Policy Decisions, 9 Blocklist Categories, 0 Client IPs Stored
- Glass-morphism or subtle gradient background

### 9. Open Source CTA Section
- "Built in the open. Owned by the community." messaging
- Two CTA buttons: Star on GitHub + Read Contributing Guide

### 10. Footer
- Logo + copyright, navigation links (GitHub, Docs, Contributing, License), tagline "Made with Go, gRPC, and paranoia."

## Technical Approach
- Single-page React app, fully responsive (mobile-first)
- Custom dark theme via Tailwind CSS variables
- Lightweight canvas-based particle animation for hero background
- Scroll-triggered animations using CSS/Intersection Observer
- Semantic HTML throughout
- SVGs and CSS for all visuals (no heavy images)
- Logo embedded from uploaded asset

