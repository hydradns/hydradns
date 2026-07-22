/*
 * ROI / cost-comparison math for the landing page calculator.
 *
 * HydraDNS is billed as a flat annual fee (₹15,000–18,000/yr) that does NOT
 * scale with device/seat count. Per-seat DNS-security competitors are priced
 * in USD per user per month, so their annual cost grows linearly with seats.
 * This module is pure (no React) so it can be unit-tested in isolation.
 */

/**
 * USD → INR assumption used for competitor pricing. Published competitor list
 * prices are in USD; this is a deliberately conservative round figure so the
 * comparison stays defensible. Surfaced in the UI next to the result.
 */
export const USD_TO_INR = 88;

/**
 * HydraDNS flat annual price in INR. Top of the ₹15,000–18,000/yr range is used
 * so the savings shown are the conservative (smallest) case.
 */
export const HYDRADNS_ANNUAL_INR = 18000;

export interface Competitor {
  id: string;
  name: string;
  /** Published list price, USD per user per month. */
  perUserPerMonthUSD: number;
  /** Short label describing the tier the price is drawn from. */
  tier: string;
}

/**
 * Approximate published list prices (USD / user / month). These are the
 * per-seat plans an Indian SMB / school / hospital would actually be quoted.
 */
export const COMPETITORS: Competitor[] = [
  { id: "cisco-umbrella", name: "Cisco Umbrella", perUserPerMonthUSD: 5.5, tier: "DNS Security Essentials" },
  { id: "cloudflare", name: "Cloudflare Gateway", perUserPerMonthUSD: 7, tier: "Zero Trust" },
  { id: "dnsfilter", name: "DNSFilter", perUserPerMonthUSD: 2.1, tier: "Basic" },
  { id: "nextdns", name: "NextDNS", perUserPerMonthUSD: 2.49, tier: "Business" },
];

export interface CompetitorResult extends Competitor {
  /** Total annual cost in INR for the given seat count. */
  annualINR: number;
  /** competitor annual − HydraDNS flat, in INR (>= 0 when competitor is dearer). */
  savingsINR: number;
  /** How many times more the competitor costs vs the HydraDNS flat fee. */
  multiplier: number;
}

export interface RoiResult {
  /** Sanitised seat count actually used for the math. */
  seats: number;
  usdToInr: number;
  hydradnsAnnualINR: number;
  competitors: CompetitorResult[];
}

export interface RoiOptions {
  usdToInr?: number;
  hydradnsAnnualINR?: number;
  competitors?: Competitor[];
}

/** Annual INR cost for a single per-seat competitor. */
export function competitorAnnualINR(
  seats: number,
  perUserPerMonthUSD: number,
  usdToInr: number = USD_TO_INR,
): number {
  const s = sanitizeSeats(seats);
  return Math.round(s * perUserPerMonthUSD * 12 * usdToInr);
}

/** Clamp seats to a non-negative integer; treat NaN / negative as 0. */
export function sanitizeSeats(seats: number): number {
  if (!Number.isFinite(seats) || seats <= 0) return 0;
  return Math.floor(seats);
}

/**
 * Given a seat/device count, return HydraDNS's flat annual cost plus each
 * competitor's per-seat annual cost, savings, and cost multiplier.
 */
export function calculateRoi(seats: number, options: RoiOptions = {}): RoiResult {
  const usdToInr = options.usdToInr ?? USD_TO_INR;
  const hydradnsAnnualINR = options.hydradnsAnnualINR ?? HYDRADNS_ANNUAL_INR;
  const competitors = options.competitors ?? COMPETITORS;
  const s = sanitizeSeats(seats);

  return {
    seats: s,
    usdToInr,
    hydradnsAnnualINR,
    competitors: competitors.map((c) => {
      const annualINR = competitorAnnualINR(s, c.perUserPerMonthUSD, usdToInr);
      return {
        ...c,
        annualINR,
        savingsINR: annualINR - hydradnsAnnualINR,
        multiplier: hydradnsAnnualINR > 0 ? annualINR / hydradnsAnnualINR : 0,
      };
    }),
  };
}

/** Format an INR amount as e.g. "₹5,80,800" (no paise). */
export function formatINR(amount: number): string {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 0,
  }).format(Math.round(amount));
}
