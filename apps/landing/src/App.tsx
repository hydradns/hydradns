import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Index from "./pages/Index";
import NotFound from "./pages/NotFound";
import DocsIndex from "./pages/docs/DocsIndex";
import GettingStarted from "./pages/docs/sections/GettingStarted";
import Installation from "./pages/docs/sections/Installation";
import Dashboard from "./pages/docs/sections/Dashboard";
import PoliciesAndBlocklists from "./pages/docs/sections/PoliciesAndBlocklists";
import Cli from "./pages/docs/sections/Cli";
import Mcp from "./pages/docs/sections/Mcp";
import Architecture from "./pages/docs/sections/Architecture";
import Troubleshooting from "./pages/docs/sections/Troubleshooting";

const queryClient = new QueryClient();

const App = () => (
  <QueryClientProvider client={queryClient}>
    <TooltipProvider>
      <Toaster />
      <Sonner />
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Index />} />
          <Route path="/docs" element={<DocsIndex />} />
          <Route path="/docs/getting-started" element={<GettingStarted />} />
          <Route path="/docs/installation" element={<Installation />} />
          <Route path="/docs/dashboard" element={<Dashboard />} />
          <Route path="/docs/policies-and-blocklists" element={<PoliciesAndBlocklists />} />
          <Route path="/docs/cli" element={<Cli />} />
          <Route path="/docs/mcp" element={<Mcp />} />
          <Route path="/docs/architecture" element={<Architecture />} />
          <Route path="/docs/troubleshooting" element={<Troubleshooting />} />
          {/* ADD ALL CUSTOM ROUTES ABOVE THE CATCH-ALL "*" ROUTE */}
          <Route path="*" element={<NotFound />} />
        </Routes>
      </BrowserRouter>
    </TooltipProvider>
  </QueryClientProvider>
);

export default App;
