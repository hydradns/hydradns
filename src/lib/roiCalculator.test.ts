import { describe, it, expect } from "vitest";
import {
  calculateRoi,
  competitorAnnualINR,
  sanitizeSeats,
  formatINR,
  HYDRADNS_ANNUAL_INR,
  USD_TO_INR,
  COMPETITORS,
} from "./roiCalculator";

describe("sanitizeSeats", () => {
  it("floors positive floats", () => {
    expect(sanitizeSeats(49.9)).toBe(49);
  });

  it("treats zero, negatives and NaN as 0", () => {
    expect(sanitizeSeats(0)).toBe(0);
    expect(sanitizeSeats(-5)).toBe(0);
    expect(sanitizeSeats(Number.NaN)).toBe(0);
    expect(sanitizeSeats(Number.POSITIVE_INFINITY)).toBe(0);
  });
});

describe("competitorAnnualINR", () => {
  it("computes seats × price × 12 months × FX", () => {
    // Cisco Umbrella @ $5.5, 100 seats, ₹88/$ = 100 * 5.5 * 12 * 88
    expect(competitorAnnualINR(100, 5.5, 88)).toBe(580800);
  });

  it("uses the default FX rate when none is given", () => {
    expect(competitorAnnualINR(10, 2.1, USD_TO_INR)).toBe(competitorAnnualINR(10, 2.1));
  });

  it("is zero for zero seats", () => {
    expect(competitorAnnualINR(0, 7)).toBe(0);
  });
});

describe("calculateRoi", () => {
  const opts = { usdToInr: 88, hydradnsAnnualINR: 18000 };

  it("returns the flat HydraDNS fee regardless of seat count", () => {
    expect(calculateRoi(10, opts).hydradnsAnnualINR).toBe(18000);
    expect(calculateRoi(1000, opts).hydradnsAnnualINR).toBe(18000);
  });

  it("computes correct per-competitor annual cost, savings and multiplier", () => {
    const cisco = calculateRoi(100, opts).competitors.find((c) => c.id === "cisco-umbrella")!;
    expect(cisco.annualINR).toBe(580800); // 100 * 5.5 * 12 * 88
    expect(cisco.savingsINR).toBe(580800 - 18000); // 562800
    expect(cisco.multiplier).toBeCloseTo(580800 / 18000, 5); // ~32.27×
  });

  it("computes DNSFilter and Cloudflare correctly for 100 seats", () => {
    const byId = Object.fromEntries(calculateRoi(100, opts).competitors.map((c) => [c.id, c]));
    expect(byId["dnsfilter"].annualINR).toBe(221760); // 100 * 2.1 * 12 * 88
    expect(byId["cloudflare"].annualINR).toBe(739200); // 100 * 7 * 12 * 88
  });

  it("returns a result for every competitor", () => {
    expect(calculateRoi(50).competitors).toHaveLength(COMPETITORS.length);
  });

  it("sanitises negative seat counts to zero cost", () => {
    const result = calculateRoi(-5, opts);
    expect(result.seats).toBe(0);
    result.competitors.forEach((c) => {
      expect(c.annualINR).toBe(0);
      expect(c.savingsINR).toBe(-18000); // still pay the (unfavourable) flat fee vs nothing
    });
  });

  it("falls back to module defaults when options are omitted", () => {
    const result = calculateRoi(100);
    expect(result.usdToInr).toBe(USD_TO_INR);
    expect(result.hydradnsAnnualINR).toBe(HYDRADNS_ANNUAL_INR);
  });
});

describe("formatINR", () => {
  it("formats whole rupees with the Indian grouping and symbol", () => {
    const formatted = formatINR(580800);
    expect(formatted).toContain("5,80,800");
    expect(formatted).toMatch(/₹|INR/);
  });
});
