import { describe, it, expect, beforeAll } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { RoiCalculatorSection } from "./RoiCalculatorSection";
import { calculateRoi, formatINR } from "@/lib/roiCalculator";

// jsdom lacks ResizeObserver (used by Radix Slider) and IntersectionObserver
// (used by the section's scroll-reveal hook); stub both so the tree mounts.
beforeAll(() => {
  if (!("ResizeObserver" in globalThis)) {
    class ResizeObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    // @ts-expect-error attach polyfill to the test global
    globalThis.ResizeObserver = ResizeObserverStub;
  }
  if (!("IntersectionObserver" in globalThis)) {
    class IntersectionObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
      takeRecords() {
        return [];
      }
    }
    // @ts-expect-error attach polyfill to the test global
    globalThis.IntersectionObserver = IntersectionObserverStub;
  }
});

describe("RoiCalculatorSection", () => {
  it("renders the section heading and the flat HydraDNS fee", () => {
    render(<RoiCalculatorSection />);
    expect(screen.getByRole("heading", { name: /flat-fee/i })).toBeInTheDocument();

    const expected = formatINR(calculateRoi(50).hydradnsAnnualINR);
    expect(screen.getByTestId("roi-hydradns-annual")).toHaveTextContent(expected);
  });

  it("shows the default (50 seat) Cisco Umbrella annual cost", () => {
    render(<RoiCalculatorSection />);
    const cisco = calculateRoi(50).competitors.find((c) => c.id === "cisco-umbrella")!;
    expect(screen.getByTestId("roi-annual-cisco-umbrella")).toHaveTextContent(
      formatINR(cisco.annualINR),
    );
  });

  it("recomputes competitor costs when the seat count changes", () => {
    render(<RoiCalculatorSection />);
    const input = screen.getByLabelText(/number of devices or seats/i);

    fireEvent.change(input, { target: { value: "200" } });

    const cisco = calculateRoi(200).competitors.find((c) => c.id === "cisco-umbrella")!;
    expect(screen.getByTestId("roi-annual-cisco-umbrella")).toHaveTextContent(
      formatINR(cisco.annualINR),
    );
    // HydraDNS flat fee must stay put as seats grow.
    expect(screen.getByTestId("roi-hydradns-annual")).toHaveTextContent(
      formatINR(calculateRoi(200).hydradnsAnnualINR),
    );
  });
});
