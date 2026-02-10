import { Navbar } from "@/components/Navbar";
import { HeroSection } from "@/components/HeroSection";
import { ProblemSection } from "@/components/ProblemSection";
import { FeaturesSection } from "@/components/FeaturesSection";
import { HowItWorksSection } from "@/components/HowItWorksSection";
import { ComparisonSection } from "@/components/ComparisonSection";
import { QuickStartSection } from "@/components/QuickStartSection";
import { StatsSection } from "@/components/StatsSection";
import { OpenSourceSection } from "@/components/OpenSourceSection";
import { Footer } from "@/components/Footer";

const Index = () => {
  return (
    <div className="min-h-screen bg-background">
      <Navbar />
      <main>
        <HeroSection />
        <ProblemSection />
        <FeaturesSection />
        <HowItWorksSection />
        <ComparisonSection />
        <QuickStartSection />
        <StatsSection />
        <OpenSourceSection />
      </main>
      <Footer />
    </div>
  );
};

export default Index;
