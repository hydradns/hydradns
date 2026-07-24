import { describe, it, expect } from "vitest";
import {
  calculateRoi,
  competitorAnnualINR,
  competitorAnnualINRForCompetitor,
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
    // Cisco Umbrella @ $3.00, 100 seats, ₹97/$ = 100 * 3.00 * 12 * 97
    expect(competitorAnnualINR(100, 3.0, 97)).toBe(349200);
  });

  it("uses the default FX rate when none is given", () => {
    expect(competitorAnnualINR(10, 1.05, USD_TO_INR)).toBe(competitorAnnualINR(10, 1.05));
  });

  it("is zero for zero seats", () => {
    expect(competitorAnnualINR(0, 7)).toBe(0);
  });
});

describe("competitorAnnualINRForCompetitor (block billing)", () => {
  const nextdns = COMPETITORS.find((c) => c.id === "nextdns")!;

  it("charges one block's worth up to and including the block size", () => {
    // 1 * $199 * ₹97 = 19,303
    expect(competitorAnnualINRForCompetitor(1, nextdns, 97)).toBe(19303);
    expect(competitorAnnualINRForCompetitor(50, nextdns, 97)).toBe(19303);
  });

  it("steps up to a second block the seat after the first block fills", () => {
    // 51 seats needs 2 blocks: 2 * $199 * ₹97 = 38,606 — not a smooth
    // per-seat increase from the 50-seat figure.
    expect(competitorAnnualINRForCompetitor(51, nextdns, 97)).toBe(38606);
  });

  it("scales in whole-block jumps for larger seat counts", () => {
    // 100 seats = 2 blocks, 150 seats = 3 blocks.
    expect(competitorAnnualINRForCompetitor(100, nextdns, 97)).toBe(2 * 199 * 97);
    expect(competitorAnnualINRForCompetitor(150, nextdns, 97)).toBe(3 * 199 * 97);
  });

  it("is zero for zero or negative seats", () => {
    expect(competitorAnnualINRForCompetitor(0, nextdns, 97)).toBe(0);
    expect(competitorAnnualINRForCompetitor(-5, nextdns, 97)).toBe(0);
  });

  it("falls back to linear per-seat pricing for competitors without a billing model", () => {
    const cloudflare = COMPETITORS.find((c) => c.id === "cloudflare")!;
    expect(competitorAnnualINRForCompetitor(100, cloudflare, 97)).toBe(
      competitorAnnualINR(100, cloudflare.perUserPerMonthUSD, 97),
    );
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
    expect(cisco.annualINR).toBe(316800); // 100 * 3.00 * 12 * 88
    expect(cisco.savingsINR).toBe(316800 - 18000); // 298800
    expect(cisco.multiplier).toBeCloseTo(316800 / 18000, 5); // ~17.6×
  });

  it("computes DNSFilter and Cloudflare correctly for 100 seats", () => {
    const byId = Object.fromEntries(calculateRoi(100, opts).competitors.map((c) => [c.id, c]));
    expect(byId["dnsfilter"].annualINR).toBe(110880); // 100 * 1.05 * 12 * 88
    expect(byId["cloudflare"].annualINR).toBe(739200); // 100 * 7 * 12 * 88
  });

  it("computes NextDNS as a step function of seat blocks, not linear per seat", () => {
    const at50 = calculateRoi(50, opts).competitors.find((c) => c.id === "nextdns")!;
    const at51 = calculateRoi(51, opts).competitors.find((c) => c.id === "nextdns")!;
    expect(at50.annualINR).toBe(1 * 199 * 88); // 1 block
    expect(at51.annualINR).toBe(2 * 199 * 88); // jumps to 2 blocks at seat 51
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
