import { Link } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";
import HeroSection from "@/components/landing/HeroSection";
import FeaturesSection from "@/components/landing/FeaturesSection";
import PricingSection from "@/components/landing/PricingSection";
import SocialProofSection from "@/components/landing/SocialProofSection";
import FooterSection from "@/components/landing/FooterSection";

const landingStructuredData = [
  {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: "Galdr",
    url: "https://pulsescore.app",
    logo: "https://pulsescore.app/og-card.svg",
    sameAs: ["https://github.com/subculture-collective/pulse-score"],
  },
  {
    "@context": "https://schema.org",
    "@type": "WebApplication",
    name: "Galdr",
    applicationCategory: "BusinessApplication",
    operatingSystem: "Web",
    offers: [
      { "@type": "Offer", name: "Free", price: "0", priceCurrency: "USD" },
      { "@type": "Offer", name: "Growth", price: "49", priceCurrency: "USD" },
      { "@type": "Offer", name: "Scale", price: "149", priceCurrency: "USD" },
    ],
    description:
      "Galdr helps B2B SaaS teams detect churn risk early with customer health intelligence.",
  },
];

export default function LandingPage() {
  return (
    <div className="galdr-noise min-h-screen bg-[var(--galdr-bg)] text-[var(--galdr-fg)]">
      <SeoMeta
        title="Galdr | Cast Customer Insight Into Action"
        description="Connect Stripe, HubSpot, and Intercom to cast retention signals into clear customer health intelligence."
        keywords={[
          "customer health score",
          "churn prevention",
          "b2b saas",
          "stripe analytics",
          "customer success software",
        ]}
        path="/"
        type="website"
        structuredData={landingStructuredData}
      />

      <header className="sticky top-0 z-20 border-b border-[var(--galdr-border)]/90 bg-[rgb(11_11_18_/_0.9)] backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-3 sm:px-10 lg:px-14">
          <Link
            to="/"
            className="text-lg font-bold tracking-tight text-[var(--galdr-fg)]"
          >
            Galdr
          </Link>

          <nav className="hidden items-center gap-6 text-sm text-[var(--galdr-fg-muted)] md:flex">
            <a
              href="#features"
              className="hover:text-[var(--galdr-accent)]"
            >
              Features
            </a>
            <a
              href="#pricing"
              className="hover:text-[var(--galdr-accent)]"
            >
              Pricing
            </a>
          </nav>

          <div className="flex items-center gap-2">
            <Link
              to="/login"
              className="galdr-button-secondary px-3 py-2 text-sm font-medium"
            >
              Sign in
            </Link>
            <Link
              to="/register"
              className="galdr-button-primary px-3 py-2 text-sm font-semibold"
            >
              Start free
            </Link>
          </div>
        </div>
      </header>

      <HeroSection />
      <FeaturesSection />
      <PricingSection />
      <SocialProofSection />
      <FooterSection />
    </div>
  );
}
